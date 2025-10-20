package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ysicing/tiga/internal/models"
	"gorm.io/gorm"
)

// AuditEventRepository 统一审计事件仓储
// 参考：.claude/specs/006-gitness-tiga/tasks.md T020
//
// 实现 audit.AuditRepository[*models.AuditEvent] 接口（位于 internal/services/audit/interface.go）
// 包含以下方法：
// - Create(ctx, event) - 创建单个审计事件
// - BatchCreate(ctx, events) - 批量创建（AsyncLogger 使用）
type AuditEventRepository interface {
	// Create 创建单个审计事件
	Create(ctx context.Context, event *models.AuditEvent) error

	// BatchCreate 批量创建审计事件（由 AsyncLogger 调用）
	BatchCreate(ctx context.Context, events []*models.AuditEvent) error

	// GetByID 根据 ID 查询单个审计事件
	GetByID(ctx context.Context, id string) (*models.AuditEvent, error)

	// List 多维度过滤查询审计事件
	// 支持的过滤条件：
	// - subsystem (string): 子系统过滤（scheduler、k8s、database 等）
	// - action (string): 操作类型过滤
	// - resource_type (string): 资源类型过滤
	// - user_uid (string): 用户 UID 过滤
	// - client_ip (string): 客户端 IP 过滤
	// - request_id (string): 请求 ID 过滤
	// - start_time (int64): 开始时间戳（毫秒）
	// - end_time (int64): 结束时间戳（毫秒）
	// - limit (int): 每页记录数
	// - offset (int): 偏移量
	List(ctx context.Context, filter map[string]interface{}) ([]*models.AuditEvent, error)

	// Count 统计符合条件的审计事件总数
	Count(ctx context.Context, filter map[string]interface{}) (int64, error)

	// DeleteOlderThan 删除指定时间之前的审计事件
	// 用于审计日志清理任务
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// auditEventRepository 审计事件仓储实现
type auditEventRepository struct {
	db *gorm.DB
}

// NewAuditEventRepository 创建审计事件仓储
func NewAuditEventRepository(db *gorm.DB) AuditEventRepository {
	return &auditEventRepository{db: db}
}

// Create 创建单个审计事件
// 实现 audit.AuditRepository[*models.AuditEvent] 接口
func (r *auditEventRepository) Create(ctx context.Context, event *models.AuditEvent) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("failed to create audit event: %w", err)
	}

	return nil
}

// BatchCreate 批量创建审计事件
// 实现 audit.AuditRepository[*models.AuditEvent] 接口
// 由 AsyncLogger 调用，实现异步批量写入
func (r *auditEventRepository) BatchCreate(ctx context.Context, events []*models.AuditEvent) error {
	if len(events) == 0 {
		return nil
	}

	// 批量验证
	for i, event := range events {
		if err := event.Validate(); err != nil {
			return fmt.Errorf("validation failed for event %d: %w", i, err)
		}
	}

	// 批量插入
	if err := r.db.WithContext(ctx).Create(events).Error; err != nil {
		return fmt.Errorf("failed to batch create audit events: %w", err)
	}

	return nil
}

// GetByID 根据 ID 查询单个审计事件
func (r *auditEventRepository) GetByID(ctx context.Context, id string) (*models.AuditEvent, error) {
	var event models.AuditEvent
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&event).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("audit event not found: id=%s", id)
		}
		return nil, fmt.Errorf("failed to get audit event: %w", err)
	}

	return &event, nil
}

// List 多维度过滤查询审计事件
// 参考：.claude/specs/006-gitness-tiga/quickstart.md 场景 4
//
// 性能要求：查询 10000 条记录应在 2 秒内完成
func (r *auditEventRepository) List(ctx context.Context, filter map[string]interface{}) ([]*models.AuditEvent, error) {
	var events []*models.AuditEvent
	query := r.db.WithContext(ctx)

	// 应用过滤条件
	query = r.applyFilters(query, filter)

	// 排序：默认按时间戳降序（最新的在前）
	query = query.Order("timestamp DESC")

	if err := query.Find(&events).Error; err != nil {
		return nil, fmt.Errorf("failed to list audit events: %w", err)
	}

	return events, nil
}

// Count 统计符合条件的审计事件总数
func (r *auditEventRepository) Count(ctx context.Context, filter map[string]interface{}) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.AuditEvent{})

	// 应用过滤条件（不包括分页）
	filterCopy := make(map[string]interface{})
	for k, v := range filter {
		if k != "limit" && k != "offset" {
			filterCopy[k] = v
		}
	}
	query = r.applyFilters(query, filterCopy)

	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count audit events: %w", err)
	}

	return count, nil
}

// DeleteOlderThan 删除指定时间之前的审计事件
// 用于审计日志清理任务（保留策略：90 天）
func (r *auditEventRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&models.AuditEvent{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old audit events: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// applyFilters 应用过滤条件到查询
// 统一的过滤逻辑，供 List 和 Count 方法使用
func (r *auditEventRepository) applyFilters(query *gorm.DB, filter map[string]interface{}) *gorm.DB {
	// 按操作类型过滤
	if action, ok := filter["action"].(string); ok && action != "" {
		query = query.Where("action = ?", action)
	}

	// 按资源类型过滤
	if resourceType, ok := filter["resource_type"].(string); ok && resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}

	// 按用户 UID 过滤
	if userUID, ok := filter["user_uid"].(string); ok && userUID != "" {
		query = query.Where("user->>'uid' = ?", userUID)
	}

	// 按客户端 IP 过滤
	if clientIP, ok := filter["client_ip"].(string); ok && clientIP != "" {
		query = query.Where("client_ip = ?", clientIP)
	}

	// 按请求 ID 过滤
	if requestID, ok := filter["request_id"].(string); ok && requestID != "" {
		query = query.Where("request_id = ?", requestID)
	}

	// 按空间路径过滤
	if spacePath, ok := filter["space_path"].(string); ok && spacePath != "" {
		query = query.Where("space_path = ?", spacePath)
	}

	// 时间范围过滤（使用 timestamp 字段）
	if startTime, ok := filter["start_time"].(int64); ok && startTime > 0 {
		query = query.Where("timestamp >= ?", startTime)
	}
	if endTime, ok := filter["end_time"].(int64); ok && endTime > 0 {
		query = query.Where("timestamp <= ?", endTime)
	}

	// 分页
	if limit, ok := filter["limit"].(int); ok && limit > 0 {
		query = query.Limit(limit)
	}
	if offset, ok := filter["offset"].(int); ok && offset > 0 {
		query = query.Offset(offset)
	}

	return query
}
