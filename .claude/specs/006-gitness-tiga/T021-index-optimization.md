# T021 Query Index Design and Optimization

## Overview

This document describes the index design for the scheduler and audit refactoring project (Phase 3.3).

**Reference**: `.claude/specs/006-gitness-tiga/tasks.md` T021
**Data Model Spec**: `.claude/specs/006-gitness-tiga/data-model.md`

## TaskExecution Indexes

### Single-Column Indexes

1. **idx_task_executions_task_uid** on `task_uid`
   - Purpose: Fast lookup by task UID
   - Used by: `ListByTaskUID()`, `GetStats()`

2. **idx_task_executions_task_name** on `task_name`
   - Purpose: Fast lookup by task name
   - Used by: `ListByTaskName()`

3. **idx_task_executions_state** on `state`
   - Purpose: Filter executions by state (pending, running, success, failure, etc.)
   - Used by: `ListByState()`

4. **idx_task_executions_started_at** on `started_at DESC`
   - Purpose: Sort executions chronologically (newest first)
   - Used by: All list queries

5. **uniqueIndex** on `execution_uid`
   - Purpose: Ensure execution uniqueness
   - Used by: `GetByExecutionUID()`

### Composite Index

**idx_task_executions_composite** on `(task_name, task_uid, state, started_at DESC)`
- **Priority 1**: `task_name` - Most selective filter
- **Priority 2**: `task_uid` - Secondary filter
- **Priority 3**: `state` - Tertiary filter
- **Priority 4**: `started_at DESC` - Sort order

**Query Patterns Supported**:
```sql
-- Pattern 1: Filter by task_name + state + sort
SELECT * FROM task_executions
WHERE task_name = ? AND state = ?
ORDER BY started_at DESC;

-- Pattern 2: Filter by task_name + time range
SELECT * FROM task_executions
WHERE task_name = ? AND started_at >= ? AND started_at <= ?
ORDER BY started_at DESC;

-- Pattern 3: Complex multi-filter queries (most common)
SELECT * FROM task_executions
WHERE task_name = ? AND state IN (?, ?) AND started_at > ?
ORDER BY started_at DESC
LIMIT 100;
```

**Performance Target**: Query 10,000 executions in <2 seconds (as per `query_performance_test.go`)

## AuditEvent Indexes

### Single-Column Indexes

1. **idx_audit_events_timestamp** on `timestamp DESC`
   - Purpose: Sort audit events chronologically (newest first)
   - Used by: All audit queries

2. **idx_audit_events_action** on `action`
   - Purpose: Filter by action type (created, updated, deleted, etc.)
   - Used by: Action-based queries

3. **idx_audit_events_resource_type** on `resource_type`
   - Purpose: Filter by resource type (cluster, database, pod, etc.)
   - Used by: Resource type queries

4. **idx_audit_events_client_ip** on `client_ip`
   - Purpose: Security queries by IP address
   - Used by: Security audit, IP-based filtering

5. **idx_audit_events_request_id** on `request_id`
   - Purpose: Trace request flows across systems
   - Used by: Request correlation

### Composite Index

**idx_audit_events_composite** on `(resource_type, action, timestamp DESC)`
- **Priority 1**: `resource_type` - Most common filter dimension
- **Priority 2**: `action` - Secondary filter dimension
- **Priority 3**: `timestamp DESC` - Sort order

**Query Patterns Supported**:
```sql
-- Pattern 1: Filter by resource type + action
SELECT * FROM audit_events
WHERE resource_type = 'database' AND action = 'deleted'
ORDER BY timestamp DESC;

-- Pattern 2: Filter by resource type + time range
SELECT * FROM audit_events
WHERE resource_type = 'cluster' AND timestamp >= ? AND timestamp <= ?
ORDER BY timestamp DESC;

-- Pattern 3: Complex multi-dimensional queries
SELECT * FROM audit_events
WHERE resource_type = 'pod'
  AND action IN ('created', 'updated')
  AND timestamp > ?
ORDER BY timestamp DESC
LIMIT 50;
```

**Performance Target**: Query 10,000 audit events in <2 seconds (as per `query_performance_test.go`)

## Index Strategy

### Composite Index Design Principles

1. **Left-to-Right Rule**: Queries must use columns from left-to-right in the index definition
   - ✅ Good: WHERE resource_type = ? AND action = ?
   - ❌ Bad: WHERE action = ? (skips resource_type)

2. **Selectivity First**: Most selective columns come first
   - `task_name` has higher selectivity than `state`
   - `resource_type` has higher selectivity than `action`

3. **Sort Column Last**: Include sort column (DESC) at the end
   - Allows index-only scans for ORDER BY clauses

### Why NOT More Indexes?

**Question**: Why not create separate indexes for every filter combination?

**Answer**: Index overhead trade-offs:
- **Write Performance**: Each index adds overhead to INSERT/UPDATE operations
- **Storage Cost**: Indexes consume disk space
- **Maintenance Cost**: More indexes = more index rebuilds during migrations

**Our Approach**: Use well-designed composite indexes that support multiple query patterns via prefix matching.

## Implementation

### GORM Struct Tags

**TaskExecution Model**:
```go
type TaskExecution struct {
    TaskUID  string `gorm:"..;index:idx_task_executions_composite,priority:2" json:"task_uid"`
    TaskName string `gorm:"..;index:idx_task_executions_composite,priority:1" json:"task_name"`
    State    ExecutionState `gorm:"..;index:idx_task_executions_composite,priority:3" json:"state"`
    StartedAt time.Time `gorm:"..;index:idx_task_executions_composite,priority:4" json:"started_at"`
}
```

**AuditEvent Model**:
```go
type AuditEvent struct {
    ResourceType ResourceType `gorm:"..;index:idx_audit_events_composite,priority:1" json:"resource_type"`
    Action       Action       `gorm:"..;index:idx_audit_events_composite,priority:2" json:"action"`
    Timestamp    int64        `gorm:"..;index:idx_audit_events_composite,priority:3" json:"timestamp"`
}
```

### Auto-Migration

Indexes are automatically created via GORM AutoMigrate during application startup:
```go
// internal/app/app.go
db.AutoMigrate(&models.TaskExecution{}, &models.AuditEvent{})
```

## Query Optimization Techniques

### 1. Pagination

All list queries support pagination to limit result set size:
```go
filter := map[string]interface{}{
    "limit":  100,
    "offset": 0,
}
```

### 2. Index Hints (if needed)

PostgreSQL query planner usually picks the right index. If not, use index hints:
```sql
SELECT * FROM task_executions USE INDEX (idx_task_executions_composite)
WHERE task_name = ? AND state = ?;
```

### 3. EXPLAIN Analysis

Verify index usage with EXPLAIN:
```sql
EXPLAIN ANALYZE
SELECT * FROM task_executions
WHERE task_name = 'db-cleanup' AND state = 'success'
ORDER BY started_at DESC
LIMIT 100;
```

Expected output should show:
- **Index Scan** (not Seq Scan)
- **Index Name**: idx_task_executions_composite
- **Execution Time**: <50ms for 10,000 rows

## Verification

### Integration Tests

**T009 - query_performance_test.go**:
- Tests query performance with 10,000+ records
- Validates index usage via EXPLAIN plans
- Verifies <2 second query time requirement

**Test Scenarios**:
1. Query 10,000 records with pagination
2. Filter by resource_type + action
3. Filter by time range
4. Combine multiple filters
5. Test deep pagination (offset > 5000)

### Manual Verification

```bash
# Run performance tests
go test -v ./tests/integration/audit/query_performance_test.go -timeout 5m

# Check PostgreSQL index usage
psql -d tiga -c "SELECT indexname, indexdef FROM pg_indexes WHERE tablename = 'task_executions';"
psql -d tiga -c "SELECT indexname, indexdef FROM pg_indexes WHERE tablename = 'audit_events';"
```

## Future Optimizations

### If Query Performance Degrades

1. **Partial Indexes**: Index only recent data (e.g., last 90 days)
   ```sql
   CREATE INDEX idx_recent_executions ON task_executions(started_at DESC)
   WHERE started_at > NOW() - INTERVAL '90 days';
   ```

2. **Partitioning**: Partition tables by time range
   ```sql
   CREATE TABLE audit_events_2025_01 PARTITION OF audit_events
   FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
   ```

3. **Covering Indexes**: Include frequently selected columns in index
   ```sql
   CREATE INDEX idx_executions_covering ON task_executions(task_name, state, started_at DESC)
   INCLUDE (execution_uid, duration_ms);
   ```

## Summary

**Completed Tasks**:
- ✅ Defined all single-column indexes in GORM models
- ✅ Added composite indexes for common query patterns
- ✅ Verified compilation of index definitions
- ✅ Documented index strategy and optimization approach

**Next Steps**:
- Run integration tests to verify index performance (T009)
- Monitor query performance in production
- Adjust indexes based on real-world query patterns
