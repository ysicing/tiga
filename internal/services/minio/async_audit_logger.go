package minio

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/audit"
)

// AsyncAuditLogger provides non-blocking audit logging for MinIO operations.
// T036: 迁移到统一 AuditEvent 模型
type AsyncAuditLogger struct {
	logger *audit.AsyncLogger[*models.AuditEvent]
}

// AsyncAuditLoggerConfig holds configuration for the async audit logger.
type AsyncAuditLoggerConfig = audit.Config

// NewAsyncAuditLogger creates an async audit logger for MinIO.
// T036: 使用统一的 AuditEventRepository
func NewAsyncAuditLogger(r repository.AuditEventRepository, config *AsyncAuditLoggerConfig) *AsyncAuditLogger {
	logger := audit.NewAsyncLogger[*models.AuditEvent](r, "MinIO", config)
	return &AsyncAuditLogger{logger: logger}
}

// LogOperation enqueues a MinIO operation for async audit logging.
// T036: 将 MinIO 特定字段序列化到 Data 字段
func (l *AsyncAuditLogger) LogOperation(
	ctx context.Context,
	instanceID uuid.UUID,
	opType, resType, resName, action, status, message string,
	operatorID *uuid.UUID,
	operatorName, clientIP string,
	details models.JSONB,
) error {
	// T036: 构建 Data 字段，包含 MinIO 特定信息
	data := map[string]string{
		"instance_id":    instanceID.String(),
		"operation_type": opType,
		"status":         status,
	}
	if message != "" {
		data["error_message"] = message
	}

	// 合并原有的 details
	if details != nil && len(details) > 0 {
		for k, v := range details {
			if strVal, ok := v.(string); ok {
				data[k] = strVal
			} else {
				jsonBytes, _ := json.Marshal(v)
				data[k] = string(jsonBytes)
			}
		}
	}

	// T036: 映射到统一 AuditEvent 模型
	entry := &models.AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now().UnixMilli(),
		Subsystem: models.SubsystemMinIO, // T036: 设置子系统为 minio
		Action:    models.Action(action),
		Resource: models.Resource{
			Type:       models.ResourceType(resType),
			Identifier: instanceID.String(),
			Data: map[string]string{
				"resource_name": resName,
			},
		},
		ClientIP: clientIP,
		Data:     data,
	}

	// 设置用户信息（如果提供）
	if operatorID != nil && operatorName != "" {
		entry.User = models.Principal{
			UID:      operatorID.String(),
			Username: operatorName,
			Type:     models.PrincipalTypeUser,
		}
	}

	return l.logger.Enqueue(entry)
}

// Shutdown gracefully shuts down the async audit logger.
func (l *AsyncAuditLogger) Shutdown(timeout time.Duration) error {
	return l.logger.Shutdown(timeout)
}

// ChannelStatus returns the current channel usage statistics.
func (l *AsyncAuditLogger) ChannelStatus() (used, capacity int) {
	return l.logger.ChannelStatus()
}
