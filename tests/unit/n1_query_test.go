package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// QueryCounter is a custom logger that counts SQL queries
type QueryCounter struct {
	logger.Interface
	QueryCount int
}

func (q *QueryCounter) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	q.QueryCount++
	if q.Interface != nil {
		q.Interface.Trace(ctx, begin, fc, err)
	}
}

func setupTestDBForN1(t *testing.T) (*gorm.DB, *QueryCounter) {
	counter := &QueryCounter{}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: counter,
	})
	require.NoError(t, err)

	// Migrate models
	err = db.AutoMigrate(
		&models.Instance{},
		&models.Alert{},
		&models.AlertEvent{},
	)
	require.NoError(t, err)

	return db, counter
}

func TestAlertRepository_ListRules_NoN1Query(t *testing.T) {
	db, counter := setupTestDBForN1(t)
	repo := repository.NewAlertRepository(db)

	// Create test instances
	instances := make([]*models.Instance, 5)
	ownerID := uuid.New()
	for i := 0; i < 5; i++ {
		instance := &models.Instance{
			Name:       "instance-" + string(rune('A'+i)),
			Type:       "mysql",
			Connection: models.JSONB{"host": "localhost", "port": 3306},
			Status:     "running",
			OwnerID:    ownerID,
		}
		err := db.Create(instance).Error
		require.NoError(t, err)
		instances[i] = instance
	}

	// Create test alert rules
	for i := 0; i < 10; i++ {
		alert := &models.Alert{
			Name:        "alert-" + string(rune('0'+i)),
			Description: "Test alert",
			InstanceID:  &instances[i%5].ID,
			RuleType:    "threshold",
			RuleConfig:  models.JSONB{},
			Severity:    "warning",
			Enabled:     true,
		}
		err := repo.CreateRule(context.Background(), alert)
		require.NoError(t, err)
	}

	// Reset counter before test query
	counter.QueryCount = 0

	// List rules with Preload (should use 2 queries: 1 for count, 1 for data with join)
	filter := &repository.ListRulesFilter{
		Page:     1,
		PageSize: 10,
	}
	rules, total, err := repo.ListRules(context.Background(), filter)

	require.NoError(t, err)
	assert.Equal(t, int64(10), total)
	assert.Len(t, rules, 10)

	// Verify no N+1 query: should be <= 3 queries (count + select + preload)
	// Without Preload, it would be 1 (count) + 1 (select) + 5 (for each distinct instance) = 7 queries
	t.Logf("Query count with Preload: %d", counter.QueryCount)
	assert.LessOrEqual(t, counter.QueryCount, 3, "Should not have N+1 queries with Preload")

	// Verify instances are loaded
	for _, rule := range rules {
		if rule.InstanceID != nil {
			assert.NotNil(t, rule.Instance, "Instance should be preloaded for rule %s", rule.Name)
			assert.NotEmpty(t, rule.Instance.Name, "Instance name should be loaded")
		}
	}
}

func TestAlertRepository_ListEnabledRules_NoN1Query(t *testing.T) {
	db, counter := setupTestDBForN1(t)
	repo := repository.NewAlertRepository(db)

	// Create test instances
	instances := make([]*models.Instance, 3)
	ownerID := uuid.New()
	for i := 0; i < 3; i++ {
		instance := &models.Instance{
			Name:       "instance-" + string(rune('X'+i)),
			Type:       "postgresql",
			Connection: models.JSONB{"host": "localhost", "port": 5432},
			Status:     "running",
			OwnerID:    ownerID,
		}
		err := db.Create(instance).Error
		require.NoError(t, err)
		instances[i] = instance
	}

	// Create enabled and disabled alert rules
	var createdAlerts []*models.Alert
	for i := 0; i < 6; i++ {
		alert := &models.Alert{
			Name:        "alert-enabled-" + string(rune('0'+i)),
			Description: "Test alert",
			InstanceID:  &instances[i%3].ID,
			RuleType:    "threshold",
			RuleConfig:  models.JSONB{},
			Severity:    "critical",
			Enabled:     true, // Create all as enabled first
		}
		err := db.Create(alert).Error
		require.NoError(t, err)
		createdAlerts = append(createdAlerts, alert)
	}

	// Now update the last 2 to disabled (workaround for GORM boolean zero-value issue)
	for i := 4; i < 6; i++ {
		err := db.Model(createdAlerts[i]).Update("enabled", false).Error
		require.NoError(t, err)
	}

	// Reset counter before test query
	counter.QueryCount = 0

	// List enabled rules with Preload
	rules, err := repo.ListEnabledRules(context.Background())

	require.NoError(t, err)
	assert.Len(t, rules, 4, "Should return only enabled rules")

	// Verify no N+1 query
	t.Logf("Query count with Preload: %d", counter.QueryCount)
	assert.LessOrEqual(t, counter.QueryCount, 3, "Should not have N+1 queries with Preload")

	// Verify instances are loaded
	for _, rule := range rules {
		assert.True(t, rule.Enabled, "All returned rules should be enabled")
		if rule.InstanceID != nil {
			assert.NotNil(t, rule.Instance, "Instance should be preloaded")
		}
	}
}

func TestAlertRepository_ListRulesByInstance_NoN1Query(t *testing.T) {
	db, counter := setupTestDBForN1(t)
	repo := repository.NewAlertRepository(db)

	ownerID := uuid.New()

	// Create test instance
	instance := &models.Instance{
		Name:       "target-instance",
		Type:       "redis",
		Connection: models.JSONB{"host": "localhost", "port": 6379},
		Status:     "running",
		OwnerID:    ownerID,
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	// Create another instance for comparison
	otherInstance := &models.Instance{
		Name:       "other-instance",
		Type:       "redis",
		Connection: models.JSONB{"host": "localhost", "port": 6380},
		Status:     "running",
		OwnerID:    ownerID,
	}
	err = db.Create(otherInstance).Error
	require.NoError(t, err)

	// Create alert rules for target instance
	for i := 0; i < 5; i++ {
		alert := &models.Alert{
			Name:        "alert-instance-" + string(rune('A'+i)),
			Description: "Test alert for target instance",
			InstanceID:  &instance.ID,
			RuleType:    "threshold",
			RuleConfig:  models.JSONB{},
			Severity:    "info",
			Enabled:     true,
		}
		err := repo.CreateRule(context.Background(), alert)
		require.NoError(t, err)
	}

	// Create alert rule for other instance
	alert := &models.Alert{
		Name:        "alert-other",
		Description: "Test alert for other instance",
		InstanceID:  &otherInstance.ID,
		RuleType:    "threshold",
		RuleConfig:  models.JSONB{},
		Severity:    "info",
		Enabled:     true,
	}
	err = repo.CreateRule(context.Background(), alert)
	require.NoError(t, err)

	// Reset counter before test query
	counter.QueryCount = 0

	// List rules by instance with Preload
	rules, err := repo.ListRulesByInstance(context.Background(), instance.ID)

	require.NoError(t, err)
	assert.Len(t, rules, 5, "Should return only rules for target instance")

	// Verify no N+1 query
	t.Logf("Query count with Preload: %d", counter.QueryCount)
	assert.LessOrEqual(t, counter.QueryCount, 2, "Should not have N+1 queries with Preload")

	// Verify instances are loaded and correct
	for _, rule := range rules {
		assert.NotNil(t, rule.Instance, "Instance should be preloaded")
		assert.Equal(t, instance.ID, rule.Instance.ID, "Should load correct instance")
		assert.Equal(t, "target-instance", rule.Instance.Name)
	}
}

func BenchmarkAlertRepository_ListRules_WithPreload(b *testing.B) {
	db, _ := setupTestDBForN1(&testing.T{})
	repo := repository.NewAlertRepository(db)

	ownerID := uuid.New()

	// Create test data
	instances := make([]*models.Instance, 10)
	for i := 0; i < 10; i++ {
		instance := &models.Instance{
			Name:       "bench-instance-" + uuid.New().String(),
			Type:       "mysql",
			Connection: models.JSONB{"host": "localhost", "port": 3306},
			Status:     "running",
			OwnerID:    ownerID,
		}
		db.Create(instance)
		instances[i] = instance
	}

	for i := 0; i < 100; i++ {
		alert := &models.Alert{
			Name:        "bench-alert-" + uuid.New().String(),
			Description: "Benchmark alert",
			InstanceID:  &instances[i%10].ID,
			RuleType:    "threshold",
			RuleConfig:  models.JSONB{},
			Severity:    "warning",
			Enabled:     true,
		}
		repo.CreateRule(context.Background(), alert)
	}

	filter := &repository.ListRulesFilter{
		Page:     1,
		PageSize: 50,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = repo.ListRules(context.Background(), filter)
	}
}
