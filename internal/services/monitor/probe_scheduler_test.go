package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/proto"
)

// mockServiceRepository is a minimal mock for testing
type mockServiceRepository struct{}

func (m *mockServiceRepository) Create(ctx context.Context, service *models.ServiceMonitor) error {
	return nil
}
func (m *mockServiceRepository) List(ctx context.Context, filter repository.ServiceFilter) ([]*models.ServiceMonitor, int64, error) {
	return nil, 0, nil
}
func (m *mockServiceRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ServiceMonitor, error) {
	return nil, nil
}
func (m *mockServiceRepository) Update(ctx context.Context, service *models.ServiceMonitor) error {
	return nil
}
func (m *mockServiceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *mockServiceRepository) SaveProbeResult(ctx context.Context, result *models.ServiceProbeResult) error {
	return nil
}
func (m *mockServiceRepository) GetProbeHistory(ctx context.Context, serviceID uuid.UUID, start, end time.Time, limit int) ([]*models.ServiceProbeResult, int64, error) {
	return nil, 0, nil
}
func (m *mockServiceRepository) GetLatestProbeResult(ctx context.Context, serviceID uuid.UUID) (*models.ServiceProbeResult, error) {
	return nil, nil
}
func (m *mockServiceRepository) GetProbeResultsByHostAndService(ctx context.Context, hostID, serviceID uuid.UUID, start, end time.Time) ([]*models.ServiceProbeResult, error) {
	return nil, nil
}
func (m *mockServiceRepository) SaveAvailability(ctx context.Context, availability *models.ServiceAvailability) error {
	return nil
}
func (m *mockServiceRepository) GetAvailability(ctx context.Context, serviceID uuid.UUID, period string, start time.Time) (*models.ServiceAvailability, error) {
	return nil, nil
}
func (m *mockServiceRepository) CalculateAvailability(ctx context.Context, serviceID uuid.UUID, start, end time.Time) (*models.ServiceAvailability, error) {
	return nil, nil
}
func (m *mockServiceRepository) SaveServiceHistory(ctx context.Context, history *models.ServiceHistory) error {
	return nil
}
func (m *mockServiceRepository) GetServiceHistories(ctx context.Context, start, end time.Time) ([]*models.ServiceHistory, error) {
	return nil, nil
}
func (m *mockServiceRepository) GetActiveHostNodes(ctx context.Context) ([]*models.HostNode, error) {
	return nil, nil
}
func (m *mockServiceRepository) GetServiceHistoryByHost(ctx context.Context, hostID uuid.UUID, start, end time.Time) ([]*models.ServiceHistory, error) {
	return nil, nil
}

// mockAgentTaskDispatcher is a minimal mock for testing
type mockAgentTaskDispatcher struct {
	queuedTasks []string
}

func (m *mockAgentTaskDispatcher) QueueTask(agentUUID string, task *proto.AgentTask) error {
	m.queuedTasks = append(m.queuedTasks, agentUUID)
	return nil
}

func (m *mockAgentTaskDispatcher) GetAllAgentUUIDs() []string {
	return []string{"agent-1", "agent-2", "agent-3"}
}

// TestNewServiceProbeScheduler tests scheduler creation
func TestNewServiceProbeScheduler(t *testing.T) {
	repo := &mockServiceRepository{}
	scheduler := NewServiceProbeScheduler(repo, nil)

	assert.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.cron)
	assert.NotNil(t, scheduler.httpClient)
	assert.NotNil(t, scheduler.sentinel)
	assert.Equal(t, repo, scheduler.serviceRepo)
}

// TestBuildCronExpression tests cron expression building for various intervals
func TestBuildCronExpression(t *testing.T) {
	scheduler := NewServiceProbeScheduler(&mockServiceRepository{}, nil)

	tests := []struct {
		name            string
		intervalSeconds int
		expectedExpr    string
	}{
		{
			name:            "5 seconds",
			intervalSeconds: 5,
			expectedExpr:    "*/5 * * * * *",
		},
		{
			name:            "10 seconds",
			intervalSeconds: 10,
			expectedExpr:    "*/10 * * * * *",
		},
		{
			name:            "30 seconds",
			intervalSeconds: 30,
			expectedExpr:    "*/30 * * * * *",
		},
		{
			name:            "60 seconds (1 minute)",
			intervalSeconds: 60,
			expectedExpr:    "0 */1 * * * *",
		},
		{
			name:            "120 seconds (2 minutes)",
			intervalSeconds: 120,
			expectedExpr:    "0 */2 * * * *",
		},
		{
			name:            "300 seconds (5 minutes)",
			intervalSeconds: 300,
			expectedExpr:    "0 */5 * * * *",
		},
		{
			name:            "600 seconds (10 minutes)",
			intervalSeconds: 600,
			expectedExpr:    "0 */10 * * * *",
		},
		{
			name:            "1800 seconds (30 minutes)",
			intervalSeconds: 1800,
			expectedExpr:    "0 */30 * * * *",
		},
		{
			name:            "3600 seconds (1 hour)",
			intervalSeconds: 3600,
			expectedExpr:    "0 0 */1 * * *",
		},
		{
			name:            "7200 seconds (2 hours)",
			intervalSeconds: 7200,
			expectedExpr:    "0 0 */2 * * *",
		},
		{
			name:            "21600 seconds (6 hours)",
			intervalSeconds: 21600,
			expectedExpr:    "0 0 */6 * * *",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := scheduler.buildCronExpression(tt.intervalSeconds)
			assert.Equal(t, tt.expectedExpr, expr)
		})
	}
}

// TestScheduleMonitor tests scheduling a monitor
func TestScheduleMonitor(t *testing.T) {
	scheduler := NewServiceProbeScheduler(&mockServiceRepository{}, nil)
	scheduler.cron.Start()
	defer scheduler.cron.Stop()

	t.Run("schedule new monitor", func(t *testing.T) {
		monitorID := uuid.New()
		monitor := &models.ServiceMonitor{
			Name:     "Test Monitor",
			Type:     models.ProbeTypeHTTP,
			Target:   "https://example.com",
			Interval: 60, // 1 minute
			Enabled:  true,
		}
		monitor.ID = monitorID // Set ID separately since it's in BaseModel

		err := scheduler.ScheduleMonitor(monitor)
		assert.NoError(t, err)

		// Verify task is stored
		task, exists := scheduler.GetTaskStatus(monitorID)
		assert.True(t, exists)
		assert.NotNil(t, task)
		assert.Equal(t, monitorID, task.MonitorID)
		assert.Equal(t, monitor.Name, task.Monitor.Name)

		// Clean up
		scheduler.UnscheduleMonitor(monitorID)
	})

	t.Run("reschedule existing monitor", func(t *testing.T) {
		monitorID := uuid.New()
		monitor := &models.ServiceMonitor{
			Name:     "Test Monitor 2",
			Type:     models.ProbeTypeTCP,
			Target:   "localhost:80",
			Interval: 30,
			Enabled:  true,
		}
		monitor.ID = monitorID

		// Schedule first time
		err := scheduler.ScheduleMonitor(monitor)
		assert.NoError(t, err)

		firstTask, _ := scheduler.GetTaskStatus(monitorID)
		firstEntryID := firstTask.CronEntryID

		// Reschedule with different interval
		monitor.Interval = 120
		err = scheduler.ScheduleMonitor(monitor)
		assert.NoError(t, err)

		secondTask, _ := scheduler.GetTaskStatus(monitorID)
		secondEntryID := secondTask.CronEntryID

		// Entry ID should be different (new schedule)
		assert.NotEqual(t, firstEntryID, secondEntryID)

		// Clean up
		scheduler.UnscheduleMonitor(monitorID)
	})
}

// TestUnscheduleMonitor tests unscheduling a monitor
func TestUnscheduleMonitor(t *testing.T) {
	scheduler := NewServiceProbeScheduler(&mockServiceRepository{}, nil)
	scheduler.cron.Start()
	defer scheduler.cron.Stop()

	t.Run("unschedule existing monitor", func(t *testing.T) {
		monitorID := uuid.New()
		monitor := &models.ServiceMonitor{
			Name:     "Test Monitor",
			Type:     models.ProbeTypeHTTP,
			Target:   "https://example.com",
			Interval: 60,
			Enabled:  true,
		}
		monitor.ID = monitorID

		// Schedule first
		err := scheduler.ScheduleMonitor(monitor)
		require.NoError(t, err)

		// Verify it exists
		_, exists := scheduler.GetTaskStatus(monitorID)
		assert.True(t, exists)

		// Unschedule
		scheduler.UnscheduleMonitor(monitorID)

		// Verify it's removed
		_, exists = scheduler.GetTaskStatus(monitorID)
		assert.False(t, exists)
	})

	t.Run("unschedule non-existent monitor", func(t *testing.T) {
		// Should not panic or error
		nonExistentID := uuid.New()
		scheduler.UnscheduleMonitor(nonExistentID)

		// Verify it doesn't exist
		_, exists := scheduler.GetTaskStatus(nonExistentID)
		assert.False(t, exists)
	})
}

// TestGetScheduledTasks tests retrieving all scheduled tasks
func TestGetScheduledTasks(t *testing.T) {
	scheduler := NewServiceProbeScheduler(&mockServiceRepository{}, nil)
	scheduler.cron.Start()
	defer scheduler.cron.Stop()

	t.Run("no tasks initially", func(t *testing.T) {
		tasks := scheduler.GetScheduledTasks()
		assert.Empty(t, tasks)
	})

	t.Run("multiple tasks", func(t *testing.T) {
		// Create multiple monitors
		monitor1 := &models.ServiceMonitor{
			Name:     "HTTP Monitor",
			Type:     models.ProbeTypeHTTP,
			Target:   "https://example.com",
			Interval: 60,
			Enabled:  true,
		}
		monitor1.ID = uuid.New()

		monitor2 := &models.ServiceMonitor{
			Name:     "TCP Monitor",
			Type:     models.ProbeTypeTCP,
			Target:   "localhost:80",
			Interval: 30,
			Enabled:  true,
		}
		monitor2.ID = uuid.New()

		monitor3 := &models.ServiceMonitor{
			Name:     "ICMP Monitor",
			Type:     models.ProbeTypeICMP,
			Target:   "8.8.8.8",
			Interval: 10,
			Enabled:  true,
		}
		monitor3.ID = uuid.New()

		monitors := []*models.ServiceMonitor{monitor1, monitor2, monitor3}

		for _, monitor := range monitors {
			err := scheduler.ScheduleMonitor(monitor)
			require.NoError(t, err)
		}

		// Get all tasks
		tasks := scheduler.GetScheduledTasks()
		assert.Len(t, tasks, 3)

		// Verify task properties
		taskMap := make(map[uuid.UUID]*ProbeTask)
		for _, task := range tasks {
			taskMap[task.MonitorID] = task
		}

		for _, monitor := range monitors {
			task, exists := taskMap[monitor.ID]
			assert.True(t, exists)
			assert.Equal(t, monitor.Name, task.Monitor.Name)
			assert.Equal(t, monitor.Type, task.Monitor.Type)
		}

		// Clean up
		for _, monitor := range monitors {
			scheduler.UnscheduleMonitor(monitor.ID)
		}
	})
}

// TestGetTaskStatus tests retrieving task status
func TestGetTaskStatus(t *testing.T) {
	scheduler := NewServiceProbeScheduler(&mockServiceRepository{}, nil)
	scheduler.cron.Start()
	defer scheduler.cron.Stop()

	t.Run("task exists", func(t *testing.T) {
		monitorID := uuid.New()
		monitor := &models.ServiceMonitor{
			Name:     "Test Monitor",
			Type:     models.ProbeTypeHTTP,
			Target:   "https://example.com",
			Interval: 60,
			Enabled:  true,
		}
		monitor.ID = monitorID

		err := scheduler.ScheduleMonitor(monitor)
		require.NoError(t, err)

		task, exists := scheduler.GetTaskStatus(monitorID)
		assert.True(t, exists)
		assert.NotNil(t, task)
		assert.Equal(t, monitorID, task.MonitorID)
		assert.Equal(t, monitor.Name, task.Monitor.Name)

		scheduler.UnscheduleMonitor(monitorID)
	})

	t.Run("task does not exist", func(t *testing.T) {
		nonExistentID := uuid.New()
		task, exists := scheduler.GetTaskStatus(nonExistentID)
		assert.False(t, exists)
		assert.Nil(t, task)
	})
}

// TestSetAgentManager tests setting agent manager
func TestSetAgentManager(t *testing.T) {
	scheduler := NewServiceProbeScheduler(&mockServiceRepository{}, nil)

	t.Run("set agent manager", func(t *testing.T) {
		mockAgent := &mockAgentTaskDispatcher{}
		scheduler.SetAgentManager(mockAgent)
		assert.NotNil(t, scheduler.agentManager)
	})

	t.Run("agent manager initially nil", func(t *testing.T) {
		newScheduler := NewServiceProbeScheduler(&mockServiceRepository{}, nil)
		assert.Nil(t, newScheduler.agentManager)
	})
}

// TestUpdateMonitorSchedule tests updating monitor schedule
func TestUpdateMonitorSchedule(t *testing.T) {
	scheduler := NewServiceProbeScheduler(&mockServiceRepository{}, nil)
	scheduler.cron.Start()
	defer scheduler.cron.Stop()

	monitorID := uuid.New()
	monitor := &models.ServiceMonitor{
		Name:     "Test Monitor",
		Type:     models.ProbeTypeHTTP,
		Target:   "https://example.com",
		Interval: 60,
		Enabled:  true,
	}
	monitor.ID = monitorID

	// Initial schedule
	err := scheduler.ScheduleMonitor(monitor)
	require.NoError(t, err)

	firstTask, _ := scheduler.GetTaskStatus(monitorID)

	// Update schedule
	monitor.Interval = 120
	err = scheduler.UpdateMonitorSchedule(monitor)
	assert.NoError(t, err)

	secondTask, _ := scheduler.GetTaskStatus(monitorID)

	// Verify schedule was updated (different entry ID)
	assert.NotEqual(t, firstTask.CronEntryID, secondTask.CronEntryID)
	assert.Equal(t, monitor.Interval, secondTask.Monitor.Interval)

	scheduler.UnscheduleMonitor(monitorID)
}

// TestSchedulerStartStop tests scheduler lifecycle
func TestSchedulerStartStop(t *testing.T) {
	t.Skip("Skipping Start/Stop test as it involves goroutines and actual cron execution - use integration tests instead")

	// This test is skipped because:
	// 1. Start() launches background goroutines
	// 2. loadAndScheduleMonitors() requires database access
	// 3. ServiceSentinel.Start() requires background processing
	// 4. Proper testing requires integration test setup
	//
	// For integration testing:
	// 1. Create scheduler with real repository
	// 2. Call Start() and verify cron is running
	// 3. Call Stop() and verify cleanup
	// 4. Verify no goroutine leaks
}

// TestProbeTaskNextRun tests that next run time is set correctly
func TestProbeTaskNextRun(t *testing.T) {
	scheduler := NewServiceProbeScheduler(&mockServiceRepository{}, nil)
	scheduler.cron.Start()
	defer scheduler.cron.Stop()

	monitorID := uuid.New()
	monitor := &models.ServiceMonitor{
		Name:     "Test Monitor",
		Type:     models.ProbeTypeHTTP,
		Target:   "https://example.com",
		Interval: 60,
		Enabled:  true,
	}
	monitor.ID = monitorID

	err := scheduler.ScheduleMonitor(monitor)
	require.NoError(t, err)

	task, exists := scheduler.GetTaskStatus(monitorID)
	assert.True(t, exists)

	// NextRun should be in the future
	assert.True(t, task.NextRun.After(time.Now()))

	// NextRun should be within reasonable range (within 2 minutes for 1-minute interval)
	assert.True(t, task.NextRun.Before(time.Now().Add(2*time.Minute)))

	scheduler.UnscheduleMonitor(monitorID)
}

// TestMultipleSchedulersIsolation tests that multiple schedulers are isolated
func TestMultipleSchedulersIsolation(t *testing.T) {
	scheduler1 := NewServiceProbeScheduler(&mockServiceRepository{}, nil)
	scheduler2 := NewServiceProbeScheduler(&mockServiceRepository{}, nil)

	scheduler1.cron.Start()
	scheduler2.cron.Start()
	defer scheduler1.cron.Stop()
	defer scheduler2.cron.Stop()

	// Schedule monitor in scheduler1
	monitorID1 := uuid.New()
	monitor1 := &models.ServiceMonitor{
		Name:     "Monitor 1",
		Type:     models.ProbeTypeHTTP,
		Target:   "https://example.com",
		Interval: 60,
		Enabled:  true,
	}
	monitor1.ID = monitorID1

	err := scheduler1.ScheduleMonitor(monitor1)
	require.NoError(t, err)

	// Verify it exists in scheduler1
	_, exists1 := scheduler1.GetTaskStatus(monitorID1)
	assert.True(t, exists1)

	// Verify it does NOT exist in scheduler2
	_, exists2 := scheduler2.GetTaskStatus(monitorID1)
	assert.False(t, exists2)

	// Verify task counts
	tasks1 := scheduler1.GetScheduledTasks()
	tasks2 := scheduler2.GetScheduledTasks()
	assert.Len(t, tasks1, 1)
	assert.Len(t, tasks2, 0)

	scheduler1.UnscheduleMonitor(monitorID1)
}
