package models

import (
	"fmt"
	"time"
)

// TaskExecution 任务执行历史记录
// 用途：记录每次任务执行的详细信息，用于历史查询、统计分析和问题排查
//
// 参考：.claude/specs/006-gitness-tiga/data-model.md 实体 2
type TaskExecution struct {
	// 基础字段
	ID       int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskUID  string `gorm:"type:varchar(255);not null;index:idx_task_executions_task_uid;index:idx_task_executions_composite,priority:2" json:"task_uid"`
	TaskName string `gorm:"type:varchar(255);not null;index:idx_task_executions_task_name;index:idx_task_executions_composite,priority:1" json:"task_name"`
	TaskType string `gorm:"type:varchar(255);not null" json:"task_type"`

	// 执行上下文
	ExecutionUID string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"execution_uid"`                                                    // UUID
	RunBy        string    `gorm:"type:varchar(255);not null" json:"run_by"`                                                                       // 实例 ID
	ScheduledAt  time.Time `gorm:"not null" json:"scheduled_at"`                                                                                   // 计划执行时间
	StartedAt    time.Time `gorm:"not null;index:idx_task_executions_started_at;index:idx_task_executions_composite,priority:4" json:"started_at"` // 实际开始时间
	FinishedAt   time.Time `gorm:"" json:"finished_at,omitempty"`                                                                                  // 实际结束时间

	// 执行结果
	State        ExecutionState `gorm:"type:varchar(32);not null;index:idx_task_executions_state;index:idx_task_executions_composite,priority:3" json:"state"`
	Result       string         `gorm:"type:text" json:"result,omitempty"`        // 执行结果数据
	ErrorMessage string         `gorm:"type:text" json:"error_message,omitempty"` // 错误信息
	ErrorStack   string         `gorm:"type:text" json:"error_stack,omitempty"`   // 错误堆栈

	// 执行指标
	DurationMs int64 `gorm:"not null;default:0" json:"duration_ms"` // 执行时长（毫秒）
	Progress   int   `gorm:"not null;default:0" json:"progress"`    // 进度（0-100）
	RetryCount int   `gorm:"not null;default:0" json:"retry_count"` // 重试次数

	// 触发方式
	TriggerType string `gorm:"type:varchar(32);not null" json:"trigger_type"` // scheduled, manual
	TriggerBy   string `gorm:"type:varchar(255)" json:"trigger_by,omitempty"` // 手动触发的用户 ID

	// 时间戳
	CreatedAt time.Time `gorm:"index" json:"created_at"`
	UpdatedAt time.Time `gorm:"index" json:"updated_at"`
}

// TableName 指定表名
func (TaskExecution) TableName() string {
	return "task_executions"
}

// ExecutionState 任务执行状态枚举
type ExecutionState string

const (
	// ExecutionStatePending 等待执行
	ExecutionStatePending ExecutionState = "pending"
	// ExecutionStateRunning 执行中
	ExecutionStateRunning ExecutionState = "running"
	// ExecutionStateSuccess 执行成功
	ExecutionStateSuccess ExecutionState = "success"
	// ExecutionStateFailure 执行失败
	ExecutionStateFailure ExecutionState = "failure"
	// ExecutionStateTimeout 超时失败
	ExecutionStateTimeout ExecutionState = "timeout"
	// ExecutionStateCancelled 已取消
	ExecutionStateCancelled ExecutionState = "cancelled"
	// ExecutionStateInterrupted 系统中断（如重启）
	ExecutionStateInterrupted ExecutionState = "interrupted"
)

// Validate 验证状态值有效性
func (s ExecutionState) Validate() error {
	switch s {
	case ExecutionStatePending, ExecutionStateRunning, ExecutionStateSuccess,
		ExecutionStateFailure, ExecutionStateTimeout, ExecutionStateCancelled,
		ExecutionStateInterrupted:
		return nil
	default:
		return fmt.Errorf("invalid execution state: %s", s)
	}
}

// String 返回字符串表示
func (s ExecutionState) String() string {
	return string(s)
}

// IsTerminal 判断是否为终态
func (s ExecutionState) IsTerminal() bool {
	switch s {
	case ExecutionStateSuccess, ExecutionStateFailure, ExecutionStateTimeout,
		ExecutionStateCancelled, ExecutionStateInterrupted:
		return true
	default:
		return false
	}
}

// IsSuccess 判断是否为成功状态
func (s ExecutionState) IsSuccess() bool {
	return s == ExecutionStateSuccess
}

// IsFailure 判断是否为失败状态（包括超时）
func (s ExecutionState) IsFailure() bool {
	switch s {
	case ExecutionStateFailure, ExecutionStateTimeout:
		return true
	default:
		return false
	}
}

// Validate 验证 TaskExecution 数据有效性
func (te *TaskExecution) Validate() error {
	// 验证必填字段
	if te.TaskUID == "" {
		return fmt.Errorf("task_uid is required")
	}
	if te.TaskName == "" {
		return fmt.Errorf("task_name is required")
	}
	if te.TaskType == "" {
		return fmt.Errorf("task_type is required")
	}
	if te.ExecutionUID == "" {
		return fmt.Errorf("execution_uid is required")
	}
	if te.RunBy == "" {
		return fmt.Errorf("run_by is required")
	}

	// 验证状态
	if err := te.State.Validate(); err != nil {
		return err
	}

	// 验证进度范围
	if te.Progress < 0 || te.Progress > 100 {
		return fmt.Errorf("progress must be between 0 and 100, got: %d", te.Progress)
	}

	// 验证时间逻辑
	if !te.StartedAt.IsZero() && !te.ScheduledAt.IsZero() {
		// ScheduledAt 可以晚于 StartedAt（手动触发立即执行）
	}

	// 验证终态时 FinishedAt 必须存在
	if te.State.IsTerminal() && te.FinishedAt.IsZero() {
		return fmt.Errorf("finished_at is required for terminal state: %s", te.State)
	}

	// 验证 FinishedAt >= StartedAt
	if !te.FinishedAt.IsZero() && !te.StartedAt.IsZero() {
		if te.FinishedAt.Before(te.StartedAt) {
			return fmt.Errorf("finished_at (%v) cannot be before started_at (%v)",
				te.FinishedAt, te.StartedAt)
		}
	}

	// 验证触发类型
	switch te.TriggerType {
	case "scheduled", "manual":
		// 有效的触发类型
	default:
		return fmt.Errorf("invalid trigger_type: %s (must be 'scheduled' or 'manual')", te.TriggerType)
	}

	return nil
}

// CalculateDuration 计算执行时长（毫秒）
func (te *TaskExecution) CalculateDuration() int64 {
	if te.StartedAt.IsZero() {
		return 0
	}

	endTime := te.FinishedAt
	if endTime.IsZero() {
		endTime = time.Now()
	}

	duration := endTime.Sub(te.StartedAt)
	return duration.Milliseconds()
}

// UpdateDuration 更新执行时长字段
func (te *TaskExecution) UpdateDuration() {
	te.DurationMs = te.CalculateDuration()
}
