package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/alert"
)

// ServiceSentinel manages service monitoring data aggregation and caching
// It maintains 30-day historical data in memory and periodically flushes to database
type ServiceSentinel struct {
	serviceRepo repository.ServiceRepository
	alertEngine *alert.AlertEngine
	mu          sync.RWMutex

	// serviceReportChannel receives probe results from agents/scheduler
	serviceReportChannel chan *ProbeReport

	// monthlyStatus stores 30-day aggregated data in memory
	// Key: ServiceMonitorID + HostNodeID composite key
	monthlyStatus map[string]*MonthlyStatus

	// pingBatch aggregates multiple ping results before DB write
	// Key: ServiceMonitorID + HostNodeID composite key
	pingBatch map[string]*PingBatch

	// statusToday tracks today's statistics
	// Key: ServiceMonitorID + HostNodeID composite key
	statusToday map[string]*TodayStats

	// running indicates if the sentinel is running
	running bool
	stopCh  chan struct{}
}

// ProbeReport represents a probe result report from agent or scheduler
type ProbeReport struct {
	ServiceMonitorID uuid.UUID
	HostNodeID       uuid.UUID // uuid.Nil for server-side probes
	Success          bool
	Latency          float32 // in milliseconds
	Timestamp        time.Time
	ErrorMessage     string
	Data             string // TLS cert info or other metadata
}

// MonthlyStatus stores 30 days of aggregated data
type MonthlyStatus struct {
	ServiceMonitorID uuid.UUID
	HostNodeID       uuid.UUID
	// Arrays for 30 days, index 0 is today, 29 is 30 days ago
	AvgDelay   [30]float32 // Average delay per day
	Up         [30]uint64  // Successful probe count per day
	Down       [30]uint64  // Failed probe count per day
	LastUpdate time.Time
}

// PingBatch accumulates ping results before writing to DB
type PingBatch struct {
	ServiceMonitorID uuid.UUID
	HostNodeID       uuid.UUID
	TotalLatency     float32
	Count            int
	Up               uint64
	Down             uint64
	LastData         string // Latest TLS cert or error
	CreatedAt        time.Time
}

// TodayStats tracks today's running statistics
type TodayStats struct {
	ServiceMonitorID uuid.UUID
	HostNodeID       uuid.UUID
	TotalLatency     float32
	Count            int
	Up               uint64
	Down             uint64
}

// NewServiceSentinel creates a new ServiceSentinel
func NewServiceSentinel(serviceRepo repository.ServiceRepository, alertEngine *alert.AlertEngine) *ServiceSentinel {
	return &ServiceSentinel{
		serviceRepo:          serviceRepo,
		alertEngine:          alertEngine,
		serviceReportChannel: make(chan *ProbeReport, 1000),
		monthlyStatus:        make(map[string]*MonthlyStatus),
		pingBatch:            make(map[string]*PingBatch),
		statusToday:          make(map[string]*TodayStats),
		stopCh:               make(chan struct{}),
	}
}

// Start starts the sentinel background workers
func (s *ServiceSentinel) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("service sentinel already running")
	}

	// Load existing 30-day data from database
	if err := s.loadMonthlyData(); err != nil {
		logrus.Errorf("Failed to load monthly data: %v", err)
	}

	s.running = true

	// Start report processing worker
	go s.reportWorker()

	// Start periodic flush worker (every 5 minutes)
	go s.flushWorker()

	// Start daily rotation worker (runs at midnight)
	go s.rotationWorker()

	logrus.Info("ServiceSentinel started")
	return nil
}

// Stop stops the sentinel
func (s *ServiceSentinel) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	close(s.stopCh)
	s.running = false

	// Flush remaining data
	s.flushAllBatches()

	logrus.Info("ServiceSentinel stopped")
}

// ReportProbeResult submits a probe result to the sentinel
func (s *ServiceSentinel) ReportProbeResult(report *ProbeReport) {
	select {
	case s.serviceReportChannel <- report:
	default:
		logrus.Warn("ServiceSentinel report channel full, dropping report")
	}
}

// reportWorker processes incoming probe reports
func (s *ServiceSentinel) reportWorker() {
	for {
		select {
		case <-s.stopCh:
			return
		case report := <-s.serviceReportChannel:
			s.processReport(report)
		}
	}
}

// processReport processes a single probe report
func (s *ServiceSentinel) processReport(report *ProbeReport) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.makeKey(report.ServiceMonitorID, report.HostNodeID)

	// Update today's stats
	today := s.statusToday[key]
	if today == nil {
		today = &TodayStats{
			ServiceMonitorID: report.ServiceMonitorID,
			HostNodeID:       report.HostNodeID,
		}
		s.statusToday[key] = today
	}

	today.Count++
	if report.Success {
		today.Up++
		today.TotalLatency += report.Latency
	} else {
		today.Down++
	}

	// Update ping batch
	batch := s.pingBatch[key]
	if batch == nil {
		batch = &PingBatch{
			ServiceMonitorID: report.ServiceMonitorID,
			HostNodeID:       report.HostNodeID,
			CreatedAt:        time.Now(),
		}
		s.pingBatch[key] = batch
	}

	batch.Count++
	if report.Success {
		batch.Up++
		batch.TotalLatency += report.Latency
	} else {
		batch.Down++
	}
	batch.LastData = report.Data

	// Evaluate alert rules for this service
	s.evaluateAlerts(report.ServiceMonitorID)

	// Flush batch if it has accumulated enough samples (e.g., 20 probes)
	if batch.Count >= 20 {
		s.flushBatch(key, batch)
	}
}

// flushBatch writes a ping batch to database
func (s *ServiceSentinel) flushBatch(key string, batch *PingBatch) {
	if batch.Count == 0 {
		return
	}

	avgDelay := float32(0)
	if batch.Up > 0 {
		avgDelay = batch.TotalLatency / float32(batch.Up)
	}

	history := &models.ServiceHistory{
		ServiceMonitorID: batch.ServiceMonitorID,
		HostNodeID:       batch.HostNodeID,
		CreatedAt:        batch.CreatedAt,
		AvgDelay:         avgDelay,
		Up:               batch.Up,
		Down:             batch.Down,
		Data:             batch.LastData,
	}

	ctx := context.Background()
	if err := s.serviceRepo.SaveServiceHistory(ctx, history); err != nil {
		logrus.Errorf("Failed to save service history: %v", err)
		return
	}

	// Update in-memory monthly status
	s.updateMonthlyStatus(key, avgDelay, batch.Up, batch.Down)

	// Reset batch
	delete(s.pingBatch, key)
}

// flushWorker periodically flushes all batches
func (s *ServiceSentinel) flushWorker() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.flushAllBatches()
		}
	}
}

// flushAllBatches flushes all pending batches to database
func (s *ServiceSentinel) flushAllBatches() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, batch := range s.pingBatch {
		s.flushBatch(key, batch)
	}
}

// rotationWorker performs daily data rotation at midnight
func (s *ServiceSentinel) rotationWorker() {
	for {
		// Calculate time until next midnight
		now := time.Now()
		tomorrow := now.Add(24 * time.Hour)
		midnight := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, now.Location())
		duration := midnight.Sub(now)

		timer := time.NewTimer(duration)

		select {
		case <-s.stopCh:
			timer.Stop()
			return
		case <-timer.C:
			s.rotateDailyData()
		}
	}
}

// rotateDailyData performs daily rotation of the 30-day data arrays
func (s *ServiceSentinel) rotateDailyData() {
	s.mu.Lock()
	defer s.mu.Unlock()

	logrus.Info("Performing daily service monitoring data rotation")

	// For each monthly status, left-shift the arrays
	for _, status := range s.monthlyStatus {
		// Shift arrays: index 0 becomes index 1, index 1 becomes index 2, etc.
		// The oldest data at index 29 is discarded
		for i := 29; i > 0; i-- {
			status.AvgDelay[i] = status.AvgDelay[i-1]
			status.Up[i] = status.Up[i-1]
			status.Down[i] = status.Down[i-1]
		}

		// Set today's data (index 0) from statusToday
		key := s.makeKey(status.ServiceMonitorID, status.HostNodeID)
		if today, ok := s.statusToday[key]; ok && today.Count > 0 {
			avgDelay := float32(0)
			if today.Up > 0 {
				avgDelay = today.TotalLatency / float32(today.Up)
			}
			status.AvgDelay[0] = avgDelay
			status.Up[0] = today.Up
			status.Down[0] = today.Down
		} else {
			// No data today, set to zeros
			status.AvgDelay[0] = 0
			status.Up[0] = 0
			status.Down[0] = 0
		}

		status.LastUpdate = time.Now()
	}

	// Clear today's stats for new day
	s.statusToday = make(map[string]*TodayStats)
}

// updateMonthlyStatus updates the in-memory monthly status for today
func (s *ServiceSentinel) updateMonthlyStatus(key string, avgDelay float32, up, down uint64) {
	status := s.monthlyStatus[key]
	if status == nil {
		// Initialize new monthly status
		status = &MonthlyStatus{
			LastUpdate: time.Now(),
		}
		s.monthlyStatus[key] = status
	}

	// Update today's data (index 0)
	// Since we're batching, we need to recalculate the average
	totalUp := status.Up[0] + up
	totalDown := status.Down[0] + down

	if totalUp > 0 {
		// Recalculate weighted average
		oldTotal := status.AvgDelay[0] * float32(status.Up[0])
		newTotal := avgDelay * float32(up)
		status.AvgDelay[0] = (oldTotal + newTotal) / float32(totalUp)
	}

	status.Up[0] = totalUp
	status.Down[0] = totalDown
	status.LastUpdate = time.Now()
}

// loadMonthlyData loads 30-day data from database into memory
func (s *ServiceSentinel) loadMonthlyData() error {
	ctx := context.Background()

	// Load service histories from the last 30 days
	startDate := time.Now().AddDate(0, 0, -30)

	histories, err := s.serviceRepo.GetServiceHistories(ctx, startDate, time.Now())
	if err != nil {
		return fmt.Errorf("failed to load service histories: %w", err)
	}

	// Group by (ServiceMonitorID, HostNodeID) and day
	type dayData struct {
		avgDelay float32
		up       uint64
		down     uint64
	}
	statusMap := make(map[string]map[int]*dayData) // key -> day index -> data

	for _, history := range histories {
		key := s.makeKey(history.ServiceMonitorID, history.HostNodeID)

		// Calculate day index (0 = today, 1 = yesterday, etc.)
		dayIndex := int(time.Since(history.CreatedAt).Hours() / 24)
		if dayIndex < 0 || dayIndex >= 30 {
			continue
		}

		if statusMap[key] == nil {
			statusMap[key] = make(map[int]*dayData)
		}

		// Aggregate data for the day
		if statusMap[key][dayIndex] == nil {
			statusMap[key][dayIndex] = &dayData{
				avgDelay: history.AvgDelay,
				up:       history.Up,
				down:     history.Down,
			}
		} else {
			// Merge multiple records for the same day
			existing := statusMap[key][dayIndex]
			totalUp := existing.up + history.Up
			if totalUp > 0 {
				existing.avgDelay = (existing.avgDelay*float32(existing.up) + history.AvgDelay*float32(history.Up)) / float32(totalUp)
			}
			existing.up = totalUp
			existing.down += history.Down
		}
	}

	// Convert to MonthlyStatus structures
	for key, dayMap := range statusMap {
		status := &MonthlyStatus{
			LastUpdate: time.Now(),
		}

		for dayIndex, data := range dayMap {
			if dayIndex < 30 {
				status.AvgDelay[dayIndex] = data.avgDelay
				status.Up[dayIndex] = data.up
				status.Down[dayIndex] = data.down
			}
		}

		s.monthlyStatus[key] = status
	}

	logrus.Infof("Loaded %d service monthly status records", len(s.monthlyStatus))
	return nil
}

// GetMonthlyStatus returns the 30-day status for a service monitor and host
func (s *ServiceSentinel) GetMonthlyStatus(serviceMonitorID, hostNodeID uuid.UUID) *MonthlyStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.makeKey(serviceMonitorID, hostNodeID)
	return s.monthlyStatus[key]
}

// makeKey creates a composite key from ServiceMonitorID and HostNodeID
func (s *ServiceSentinel) makeKey(serviceMonitorID, hostNodeID uuid.UUID) string {
	return fmt.Sprintf("%s:%s", serviceMonitorID.String(), hostNodeID.String())
}

// LoadStats returns the internal monthly status map (direct reference, not thread-safe for writing)
// This method is used for internal access where performance is critical
func (s *ServiceSentinel) LoadStats() map[string]*MonthlyStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.monthlyStatus
}

// CopyStats creates a thread-safe copy of statistics for external use
// Returns aggregated data enriched with service monitor names and calculated fields
func (s *ServiceSentinel) CopyStats(ctx context.Context) (map[string]*ServiceResponseItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*ServiceResponseItem)

	// Get all service monitors to populate names
	monitors, _, err := s.serviceRepo.List(ctx, repository.ServiceFilter{
		Page:     1,
		PageSize: 10000, // Get all
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load service monitors: %w", err)
	}

	// Create monitor ID to name mapping
	monitorNames := make(map[uuid.UUID]string)
	for _, mon := range monitors {
		monitorNames[mon.ID] = mon.Name
	}

	// Build response items
	// Group by ServiceMonitorID (aggregate across all hosts)
	serviceStats := make(map[uuid.UUID]*ServiceResponseItem)

	for _, status := range s.monthlyStatus {
		serviceID := status.ServiceMonitorID

		if serviceStats[serviceID] == nil {
			serviceStats[serviceID] = &ServiceResponseItem{
				ServiceMonitorID: serviceID,
				ServiceName:      monitorNames[serviceID],
			}
		}

		item := serviceStats[serviceID]

		// Aggregate 30-day data
		for i := 0; i < 30; i++ {
			item.Up[i] += status.Up[i]
			item.Down[i] += status.Down[i]

			// Calculate weighted average delay
			if status.Up[i] > 0 {
				oldTotal := item.Delay[i] * float32(item.Up[i]-status.Up[i])
				newTotal := status.AvgDelay[i] * float32(status.Up[i])
				if (item.Up[i]) > 0 {
					item.Delay[i] = (oldTotal + newTotal) / float32(item.Up[i])
				}
			}
		}
	}

	// Calculate aggregated statistics and status code
	for serviceID, item := range serviceStats {
		// Calculate total up/down
		var totalUp, totalDown uint64
		for i := 0; i < 30; i++ {
			totalUp += item.Up[i]
			totalDown += item.Down[i]
		}
		item.TotalUp = totalUp
		item.TotalDown = totalDown

		// Calculate uptime percentage
		totalChecks := totalUp + totalDown
		if totalChecks > 0 {
			item.UptimePercentage = float64(totalUp) / float64(totalChecks) * 100
		}

		// Determine status code
		item.StatusCode = getStatusCodeString(item.UptimePercentage)

		// Get today's statistics from statusToday
		for key, todayStats := range s.statusToday {
			if todayStats.ServiceMonitorID == serviceID {
				item.CurrentUp += todayStats.Up
				item.CurrentDown += todayStats.Down
			}
			_ = key // Suppress unused variable warning
		}

		// Store in result map using service ID as key
		result[serviceID.String()] = item
	}

	return result, nil
}

// getStatusCodeString returns the status code string based on uptime percentage
func getStatusCodeString(uptimePercent float64) string {
	if uptimePercent == 0 {
		return "Unknown"
	}
	if uptimePercent >= 95.0 {
		return "Good"
	}
	if uptimePercent >= 80.0 {
		return "LowAvailability"
	}
	return "Down"
}

// evaluateAlerts evaluates alert rules for a service monitor
func (s *ServiceSentinel) evaluateAlerts(serviceMonitorID uuid.UUID) {
	if s.alertEngine == nil {
		return
	}

	// Calculate current availability for this service
	var totalUp, totalDown uint64
	var totalLatency float32

	// Aggregate across all hosts for this service
	for key, todayStats := range s.statusToday {
		if todayStats.ServiceMonitorID == serviceMonitorID {
			totalUp += todayStats.Up
			totalDown += todayStats.Down
			totalLatency += todayStats.TotalLatency
		}
		_ = key // Suppress unused variable warning
	}

	totalChecks := totalUp + totalDown
	if totalChecks == 0 {
		return // No data to evaluate
	}

	uptimePercentage := float64(totalUp) / float64(totalChecks) * 100
	avgLatency := float32(0)
	if totalUp > 0 {
		avgLatency = totalLatency / float32(totalUp)
	}

	// Create availability snapshot for alert evaluation
	availability := &models.ServiceAvailability{
		ServiceMonitorID: serviceMonitorID,
		Period:           "realtime",
		StartTime:        time.Now().Add(-time.Minute),
		EndTime:          time.Now(),
		TotalChecks:      int(totalChecks),
		SuccessfulChecks: int(totalUp),
		FailedChecks:     int(totalDown),
		UptimePercentage: uptimePercentage,
		AvgLatency:       float64(avgLatency),
		MinLatency:       0,
		MaxLatency:       0,
		DowntimeSeconds:  0,
	}

	// Evaluate alert rules
	ctx := context.Background()
	if err := s.alertEngine.EvaluateServiceRules(ctx, serviceMonitorID, availability); err != nil {
		logrus.Errorf("Failed to evaluate service alert rules for %s: %v", serviceMonitorID, err)
	}
}
