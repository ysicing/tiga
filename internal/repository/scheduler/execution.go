package scheduler

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// ExecutionRepository 任务执行历史仓储接口
type ExecutionRepository interface {
	// Create 创建执行记录
	Create(ctx context.Context, execution *models.TaskExecution) error

	// Update 更新执行记录
	Update(ctx context.Context, execution *models.TaskExecution) error

	// GetByID 根据 ID 查询执行记录
	GetByID(ctx context.Context, id int64) (*models.TaskExecution, error)

	// GetByExecutionUID 根据执行 UID 查询
	GetByExecutionUID(ctx context.Context, executionUID string) (*models.TaskExecution, error)

	// ListByTaskUID 查询任务的所有执行历史
	ListByTaskUID(ctx context.Context, taskUID string, limit, offset int) ([]*models.TaskExecution, error)

	// ListByTaskName 根据任务名称查询执行历史
	ListByTaskName(ctx context.Context, taskName string, limit, offset int) ([]*models.TaskExecution, error)

	// ListByState 根据状态查询执行历史
	ListByState(ctx context.Context, state models.ExecutionState, limit, offset int) ([]*models.TaskExecution, error)

	// List 查询执行历史（支持过滤和分页）
	List(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]*models.TaskExecution, error)

	// Count 统计执行记录总数
	Count(ctx context.Context, filter map[string]interface{}) (int64, error)

	// GetStats 获取任务统计数据
	GetStats(ctx context.Context, taskUID string) (*TaskStats, error)

	// GetGlobalStats 获取全局统计数据
	GetGlobalStats(ctx context.Context) (*GlobalStats, error)

	// DeleteOlderThan 删除指定时间之前的执行记录
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// TaskStats 单任务统计数据
type TaskStats struct {
	TaskUID            string
	TaskName           string
	TotalExecutions    int64
	SuccessExecutions  int64
	FailureExecutions  int64
	TimeoutExecutions  int64
	AverageDurationMs  int64
	LastExecutedAt     time.Time
	LastExecutionState models.ExecutionState
}

// GlobalStats 全局统计数据
type GlobalStats struct {
	TotalTasks        int64
	EnabledTasks      int64
	TotalExecutions   int64
	SuccessExecutions int64
	FailureExecutions int64
	SuccessRate       float64
	AverageDurationMs int64
	TaskStats         []*TaskStats
}

// taskExecutionRepository TaskExecution 仓储实现
type taskExecutionRepository struct {
	db *gorm.DB
}

// NewExecutionRepository 创建 TaskExecution 仓储
func NewExecutionRepository(db *gorm.DB) ExecutionRepository {
	return &taskExecutionRepository{db: db}
}

// Create 创建执行记录
func (r *taskExecutionRepository) Create(ctx context.Context, execution *models.TaskExecution) error {
	if err := execution.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(execution).Error; err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	return nil
}

// Update 更新执行记录
func (r *taskExecutionRepository) Update(ctx context.Context, execution *models.TaskExecution) error {
	if err := execution.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := r.db.WithContext(ctx).Save(execution).Error; err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	return nil
}

// GetByID 根据 ID 查询执行记录
func (r *taskExecutionRepository) GetByID(ctx context.Context, id int64) (*models.TaskExecution, error) {
	var execution models.TaskExecution
	if err := r.db.WithContext(ctx).First(&execution, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("execution not found: id=%d", id)
		}
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	return &execution, nil
}

// GetByExecutionUID 根据执行 UID 查询
func (r *taskExecutionRepository) GetByExecutionUID(ctx context.Context, executionUID string) (*models.TaskExecution, error) {
	var execution models.TaskExecution
	if err := r.db.WithContext(ctx).
		Where("execution_uid = ?", executionUID).
		First(&execution).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("execution not found: execution_uid=%s", executionUID)
		}
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	return &execution, nil
}

// ListByTaskUID 查询任务的所有执行历史
func (r *taskExecutionRepository) ListByTaskUID(ctx context.Context, taskUID string, limit, offset int) ([]*models.TaskExecution, error) {
	var executions []*models.TaskExecution
	query := r.db.WithContext(ctx).
		Where("task_uid = ?", taskUID).
		Order("started_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}

	return executions, nil
}

// ListByTaskName 根据任务名称查询执行历史
func (r *taskExecutionRepository) ListByTaskName(ctx context.Context, taskName string, limit, offset int) ([]*models.TaskExecution, error) {
	var executions []*models.TaskExecution
	query := r.db.WithContext(ctx).
		Where("task_name = ?", taskName).
		Order("started_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}

	return executions, nil
}

// ListByState 根据状态查询执行历史
func (r *taskExecutionRepository) ListByState(ctx context.Context, state models.ExecutionState, limit, offset int) ([]*models.TaskExecution, error) {
	var executions []*models.TaskExecution
	query := r.db.WithContext(ctx).
		Where("state = ?", state).
		Order("started_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}

	return executions, nil
}

// List 查询执行历史（支持过滤和分页）
// T022: Updated to accept limit and offset as separate parameters
func (r *taskExecutionRepository) List(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]*models.TaskExecution, error) {
	var executions []*models.TaskExecution
	query := r.db.WithContext(ctx)

	// 应用过滤条件
	if taskName, ok := filter["task_name"].(string); ok && taskName != "" {
		query = query.Where("task_name = ?", taskName)
	}
	if taskUID, ok := filter["task_uid"].(string); ok && taskUID != "" {
		query = query.Where("task_uid = ?", taskUID)
	}
	if state, ok := filter["state"].(models.ExecutionState); ok {
		query = query.Where("state = ?", state)
	}
	if triggerType, ok := filter["trigger_type"].(string); ok && triggerType != "" {
		query = query.Where("trigger_type = ?", triggerType)
	}

	// 时间范围过滤
	if startTime, ok := filter["start_time"].(time.Time); ok {
		query = query.Where("started_at >= ?", startTime)
	}
	if endTime, ok := filter["end_time"].(time.Time); ok {
		query = query.Where("started_at <= ?", endTime)
	}

	// 分页 (use provided parameters instead of filter)
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// 排序
	query = query.Order("started_at DESC")

	if err := query.Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}

	return executions, nil
}

// Count 统计执行记录总数
func (r *taskExecutionRepository) Count(ctx context.Context, filter map[string]interface{}) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.TaskExecution{})

	// 应用过滤条件（同 List 方法）
	if taskName, ok := filter["task_name"].(string); ok && taskName != "" {
		query = query.Where("task_name = ?", taskName)
	}
	if taskUID, ok := filter["task_uid"].(string); ok && taskUID != "" {
		query = query.Where("task_uid = ?", taskUID)
	}
	if state, ok := filter["state"].(string); ok && state != "" {
		query = query.Where("state = ?", state)
	}
	if triggerType, ok := filter["trigger_type"].(string); ok && triggerType != "" {
		query = query.Where("trigger_type = ?", triggerType)
	}

	// 时间范围过滤
	if startTime, ok := filter["start_time"].(int64); ok && startTime > 0 {
		query = query.Where("EXTRACT(EPOCH FROM started_at) * 1000 >= ?", startTime)
	}
	if endTime, ok := filter["end_time"].(int64); ok && endTime > 0 {
		query = query.Where("EXTRACT(EPOCH FROM started_at) * 1000 <= ?", endTime)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count executions: %w", err)
	}

	return count, nil
}

// GetStats 获取任务统计数据
func (r *taskExecutionRepository) GetStats(ctx context.Context, taskUID string) (*TaskStats, error) {
	var stats TaskStats
	stats.TaskUID = taskUID

	// 查询基本信息
	var execution models.TaskExecution
	if err := r.db.WithContext(ctx).
		Where("task_uid = ?", taskUID).
		Order("started_at DESC").
		First(&execution).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no execution found for task: %s", taskUID)
		}
		return nil, fmt.Errorf("failed to get task info: %w", err)
	}
	stats.TaskName = execution.TaskName
	stats.LastExecutedAt = execution.StartedAt
	stats.LastExecutionState = execution.State

	// 统计总数
	if err := r.db.WithContext(ctx).
		Model(&models.TaskExecution{}).
		Where("task_uid = ?", taskUID).
		Count(&stats.TotalExecutions).Error; err != nil {
		return nil, fmt.Errorf("failed to count total executions: %w", err)
	}

	// 统计成功数
	if err := r.db.WithContext(ctx).
		Model(&models.TaskExecution{}).
		Where("task_uid = ? AND state = ?", taskUID, models.ExecutionStateSuccess).
		Count(&stats.SuccessExecutions).Error; err != nil {
		return nil, fmt.Errorf("failed to count success executions: %w", err)
	}

	// 统计失败数
	if err := r.db.WithContext(ctx).
		Model(&models.TaskExecution{}).
		Where("task_uid = ? AND state = ?", taskUID, models.ExecutionStateFailure).
		Count(&stats.FailureExecutions).Error; err != nil {
		return nil, fmt.Errorf("failed to count failure executions: %w", err)
	}

	// 统计超时数
	if err := r.db.WithContext(ctx).
		Model(&models.TaskExecution{}).
		Where("task_uid = ? AND state = ?", taskUID, models.ExecutionStateTimeout).
		Count(&stats.TimeoutExecutions).Error; err != nil {
		return nil, fmt.Errorf("failed to count timeout executions: %w", err)
	}

	// 计算平均执行时长
	var avgDuration *float64
	if err := r.db.WithContext(ctx).
		Model(&models.TaskExecution{}).
		Where("task_uid = ? AND state = ?", taskUID, models.ExecutionStateSuccess).
		Select("AVG(duration_ms)").
		Scan(&avgDuration).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate average duration: %w", err)
	}
	if avgDuration != nil {
		stats.AverageDurationMs = int64(*avgDuration)
	}

	return &stats, nil
}

// GetGlobalStats 获取全局统计数据
func (r *taskExecutionRepository) GetGlobalStats(ctx context.Context) (*GlobalStats, error) {
	var stats GlobalStats

	// 统计任务总数（从 task_executions 表中去重 task_uid）
	var uniqueTaskUIDs []string
	if err := r.db.WithContext(ctx).
		Model(&models.TaskExecution{}).
		Distinct("task_uid").
		Pluck("task_uid", &uniqueTaskUIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to count unique tasks: %w", err)
	}
	stats.TotalTasks = int64(len(uniqueTaskUIDs))

	// TODO: 启用的任务数需要从 scheduled_tasks 表查询（待 ScheduledTask 模型创建后实现）
	stats.EnabledTasks = 0

	// 统计执行总数
	if err := r.db.WithContext(ctx).
		Model(&models.TaskExecution{}).
		Count(&stats.TotalExecutions).Error; err != nil {
		return nil, fmt.Errorf("failed to count total executions: %w", err)
	}

	// 统计成功数
	if err := r.db.WithContext(ctx).
		Model(&models.TaskExecution{}).
		Where("state = ?", models.ExecutionStateSuccess).
		Count(&stats.SuccessExecutions).Error; err != nil {
		return nil, fmt.Errorf("failed to count success executions: %w", err)
	}

	// 统计失败数（包括 failure 和 timeout）
	if err := r.db.WithContext(ctx).
		Model(&models.TaskExecution{}).
		Where("state IN ?", []models.ExecutionState{models.ExecutionStateFailure, models.ExecutionStateTimeout}).
		Count(&stats.FailureExecutions).Error; err != nil {
		return nil, fmt.Errorf("failed to count failure executions: %w", err)
	}

	// 计算成功率
	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(stats.SuccessExecutions) / float64(stats.TotalExecutions) * 100
	}

	// 计算平均执行时长
	var avgDuration *float64
	if err := r.db.WithContext(ctx).
		Model(&models.TaskExecution{}).
		Where("state = ?", models.ExecutionStateSuccess).
		Select("AVG(duration_ms)").
		Scan(&avgDuration).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate average duration: %w", err)
	}
	if avgDuration != nil {
		stats.AverageDurationMs = int64(*avgDuration)
	}

	// 获取每个任务的统计数据
	var taskUIDs []string
	if err := r.db.WithContext(ctx).
		Model(&models.TaskExecution{}).
		Distinct("task_uid").
		Pluck("task_uid", &taskUIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to get task UIDs: %w", err)
	}

	for _, taskUID := range taskUIDs {
		taskStats, err := r.GetStats(ctx, taskUID)
		if err != nil {
			// 跳过错误的任务
			continue
		}
		stats.TaskStats = append(stats.TaskStats, taskStats)
	}

	return &stats, nil
}

// DeleteOlderThan 删除指定时间之前的执行记录
func (r *taskExecutionRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&models.TaskExecution{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old executions: %w", result.Error)
	}

	return result.RowsAffected, nil
}
