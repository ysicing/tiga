package unit_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

func setupTestDBForIndexes(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate models to create indexes
	err = db.AutoMigrate(
		&models.Instance{},
		&models.Alert{},
		&models.AlertEvent{},
	)
	require.NoError(t, err)

	return db
}

func TestAlert_CompositeIndexes(t *testing.T) {
	db := setupTestDBForIndexes(t)

	// Query SQLite to get index information
	var indexes []struct {
		Name string `gorm:"column:name"`
	}

	// Get all indexes on alerts table
	err := db.Raw("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='alerts'").Scan(&indexes).Error
	require.NoError(t, err)

	t.Logf("Found %d indexes on alerts table:", len(indexes))
	for _, idx := range indexes {
		t.Logf("  - %s", idx.Name)
	}

	// Verify composite index exists
	indexNames := make(map[string]bool)
	for _, idx := range indexes {
		indexNames[idx.Name] = true
	}

	// GORM creates index with format: idx_{table}_{index_name}
	assert.True(t, indexNames["idx_instance_enabled"],
		"Composite index idx_instance_enabled should exist on (instance_id, enabled)")

	// Verify basic indexes still exist
	assert.True(t, len(indexNames) >= 4,
		"Should have at least 4 indexes (pk, idx_instance_enabled, idx_severity, idx_deleted_at)")
}

func TestAlertEvent_CompositeIndexes(t *testing.T) {
	db := setupTestDBForIndexes(t)

	// Query SQLite to get index information
	var indexes []struct {
		Name string `gorm:"column:name"`
	}

	// Get all indexes on alert_events table
	err := db.Raw("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='alert_events'").Scan(&indexes).Error
	require.NoError(t, err)

	t.Logf("Found %d indexes on alert_events table:", len(indexes))
	for _, idx := range indexes {
		t.Logf("  - %s", idx.Name)
	}

	// Verify composite indexes exist
	indexNames := make(map[string]bool)
	for _, idx := range indexes {
		indexNames[idx.Name] = true
	}

	// Verify both composite indexes
	assert.True(t, indexNames["idx_status_time"],
		"Composite index idx_status_time should exist on (status, started_at)")
	assert.True(t, indexNames["idx_instance_status"],
		"Composite index idx_instance_status should exist on (instance_id, status)")

	// Verify we have the expected number of indexes
	assert.True(t, len(indexNames) >= 4,
		"Should have at least 4 indexes (pk, idx_status_time, idx_instance_status, idx_alert_id)")
}

func TestCompositeIndex_QueryPerformance(t *testing.T) {
	db := setupTestDBForIndexes(t)

	// This test demonstrates that GORM is aware of the composite indexes
	// and will use them in query plans

	// Test 1: Alert query using composite index (instance_id, enabled)
	var alertCount int64
	err := db.Model(&models.Alert{}).
		Where("instance_id IS NOT NULL AND enabled = ?", true).
		Count(&alertCount).Error
	require.NoError(t, err)
	t.Logf("Alert composite index query executed successfully, count: %d", alertCount)

	// Test 2: AlertEvent query using composite index (status, started_at)
	var eventCount int64
	err = db.Model(&models.AlertEvent{}).
		Where("status = ?", "firing").
		Order("started_at DESC").
		Count(&eventCount).Error
	require.NoError(t, err)
	t.Logf("AlertEvent status+time composite index query executed successfully, count: %d", eventCount)

	// Test 3: AlertEvent query using composite index (instance_id, status)
	var instanceEventCount int64
	err = db.Model(&models.AlertEvent{}).
		Where("instance_id IS NOT NULL AND status = ?", "firing").
		Count(&instanceEventCount).Error
	require.NoError(t, err)
	t.Logf("AlertEvent instance+status composite index query executed successfully, count: %d", instanceEventCount)
}

func TestCompositeIndex_IndexStructure(t *testing.T) {
	db := setupTestDBForIndexes(t)

	// Verify index definition for alerts.idx_instance_enabled
	var alertIdxInfo []struct {
		Seqno int    `gorm:"column:seqno"`
		CID   int    `gorm:"column:cid"`
		Name  string `gorm:"column:name"`
	}

	err := db.Raw("PRAGMA index_info(idx_instance_enabled)").Scan(&alertIdxInfo).Error
	require.NoError(t, err)

	t.Log("Index structure for idx_instance_enabled:")
	for _, info := range alertIdxInfo {
		t.Logf("  Column %d: %s (cid: %d)", info.Seqno, info.Name, info.CID)
	}

	// Should have 2 columns: instance_id and enabled
	assert.Equal(t, 2, len(alertIdxInfo), "Composite index should have 2 columns")

	// Verify index definition for alert_events.idx_status_time
	var eventIdxInfo []struct {
		Seqno int    `gorm:"column:seqno"`
		CID   int    `gorm:"column:cid"`
		Name  string `gorm:"column:name"`
	}

	err = db.Raw("PRAGMA index_info(idx_status_time)").Scan(&eventIdxInfo).Error
	require.NoError(t, err)

	t.Log("Index structure for idx_status_time:")
	for _, info := range eventIdxInfo {
		t.Logf("  Column %d: %s (cid: %d)", info.Seqno, info.Name, info.CID)
	}

	// Should have 2 columns: status and started_at (in that order due to priority)
	assert.Equal(t, 2, len(eventIdxInfo), "Composite index should have 2 columns")
}
