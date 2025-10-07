package monitor

import (
	"context"
	"time"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// ServiceProbeService handles business logic for service monitoring
type ServiceProbeService struct {
	serviceRepo repository.ServiceRepository
	scheduler   *ServiceProbeScheduler
}

// NewServiceProbeService creates a new service probe service
func NewServiceProbeService(serviceRepo repository.ServiceRepository, scheduler *ServiceProbeScheduler) *ServiceProbeService {
	return &ServiceProbeService{
		serviceRepo: serviceRepo,
		scheduler:   scheduler,
	}
}

// CreateMonitor creates a new service monitor and schedules it
func (s *ServiceProbeService) CreateMonitor(ctx context.Context, monitor *models.ServiceMonitor) error {
	if err := s.serviceRepo.Create(ctx, monitor); err != nil {
		return err
	}

	// Schedule the monitor if enabled
	if monitor.Enabled {
		return s.scheduler.ScheduleMonitor(monitor)
	}
	return nil
}

// UpdateMonitor updates a monitor and reschedules if needed
func (s *ServiceProbeService) UpdateMonitor(ctx context.Context, monitor *models.ServiceMonitor) error {
	if err := s.serviceRepo.Update(ctx, monitor); err != nil {
		return err
	}

	// Update schedule
	if monitor.Enabled {
		return s.scheduler.UpdateMonitorSchedule(monitor)
	} else {
		s.scheduler.UnscheduleMonitor(monitor.ID)
	}
	return nil
}

// DeleteMonitor deletes a monitor and unschedules it
func (s *ServiceProbeService) DeleteMonitor(ctx context.Context, id uint) error {
	s.scheduler.UnscheduleMonitor(id)
	return s.serviceRepo.Delete(ctx, id)
}

// GetMonitor retrieves a monitor with latest status
func (s *ServiceProbeService) GetMonitor(ctx context.Context, id uint) (*models.ServiceMonitor, error) {
	monitor, err := s.serviceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Enrich with latest probe result
	if latest, err := s.serviceRepo.GetLatestProbeResult(ctx, id); err == nil {
		if latest.Success {
			monitor.Status = "online"
		} else {
			monitor.Status = "offline"
		}
		monitor.LastCheckTime = &latest.Timestamp
	}

	return monitor, nil
}

// GetAvailabilityStats calculates availability statistics for a period
func (s *ServiceProbeService) GetAvailabilityStats(ctx context.Context, monitorID uint, period string) (*models.ServiceAvailability, error) {
	now := time.Now()
	var start time.Time

	switch period {
	case "1h":
		start = now.Add(-1 * time.Hour)
	case "24h":
		start = now.Add(-24 * time.Hour)
	case "7d":
		start = now.Add(-7 * 24 * time.Hour)
	case "30d":
		start = now.Add(-30 * 24 * time.Hour)
	default:
		start = now.Add(-24 * time.Hour)
		period = "24h"
	}

	// Calculate from probe results
	availability, err := s.serviceRepo.CalculateAvailability(ctx, monitorID, start, now)
	if err != nil {
		return nil, err
	}

	availability.Period = period
	return availability, nil
}

// TriggerManualProbe triggers a manual probe execution
func (s *ServiceProbeService) TriggerManualProbe(ctx context.Context, monitorID uint) (*models.ServiceProbeResult, error) {
	return s.scheduler.TriggerManualProbe(ctx, monitorID)
}
