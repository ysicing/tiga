package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/audit"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

const contextKeyClientIP contextKey = "client_ip"

// AuditEntry represents an audit log entry for database operations
type AuditEntry struct {
	InstanceID *uuid.UUID
	Operator   string
	Action     string
	TargetType string
	TargetName string
	Details    map[string]interface{}
	Success    bool
	Error      error
	ClientIP   string
}

// AsyncAuditLogger provides non-blocking audit logging with batching for database operations.
// T037: 迁移到统一 AuditEvent 模型
type AsyncAuditLogger struct {
	logger *audit.AsyncLogger[*models.AuditEvent]
}

// AsyncAuditLoggerConfig holds configuration for the async audit logger.
type AsyncAuditLoggerConfig = audit.Config

// NewAsyncAuditLogger creates an async audit logger with the given configuration.
// T037: 使用统一的 AuditEventRepository
func NewAsyncAuditLogger(repo repository.AuditEventRepository, config *AsyncAuditLoggerConfig) *AsyncAuditLogger {
	logger := audit.NewAsyncLogger[*models.AuditEvent](repo, "Database", config)
	return &AsyncAuditLogger{logger: logger}
}

// LogAction enqueues an audit entry for async processing.
// T037: 将 Database 特定字段序列化到 Data 字段
func (l *AsyncAuditLogger) LogAction(ctx context.Context, entry AuditEntry) error {
	if entry.Operator == "" {
		return fmt.Errorf("operator is required")
	}
	if entry.Action == "" {
		return fmt.Errorf("action is required")
	}

	clientIP := entry.ClientIP
	if clientIP == "" {
		clientIP = ExtractClientIP(ctx)
	}

	// T037: 构建 Data 字段，包含 Database 特定信息
	data := map[string]string{
		"target_type": entry.TargetType,
		"target_name": entry.TargetName,
		"success":     fmt.Sprintf("%t", entry.Success),
	}

	// 添加实例 ID（如果提供）
	if entry.InstanceID != nil {
		data["instance_id"] = entry.InstanceID.String()
	}

	// 添加错误信息（如果有）
	if entry.Error != nil {
		data["error_message"] = entry.Error.Error()
	}

	// 合并原有的 details
	if entry.Details != nil {
		detailsJSON, err := marshalDetails(entry.Details)
		if err == nil && detailsJSON != "" {
			var detailsMap map[string]interface{}
			if err := json.Unmarshal([]byte(detailsJSON), &detailsMap); err == nil {
				for k, v := range detailsMap {
					if strVal, ok := v.(string); ok {
						data[k] = strVal
					} else {
						jsonBytes, _ := json.Marshal(v)
						data[k] = string(jsonBytes)
					}
				}
			}
		}
	}

	// T037: 映射到统一 AuditEvent 模型
	auditEvent := &models.AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now().UnixMilli(),
		Subsystem: models.SubsystemDatabase, // T037: 设置子系统为 database
		Action:    models.Action(entry.Action),
		Resource: models.Resource{
			Type: models.ResourceType(entry.TargetType),
			Data: map[string]string{
				"target_name": entry.TargetName,
			},
		},
		User: models.Principal{
			Username: entry.Operator,
			Type:     models.PrincipalTypeUser,
		},
		ClientIP: clientIP,
		Data:     data,
	}

	// 如果有实例 ID，设置 Resource.Identifier
	if entry.InstanceID != nil {
		auditEvent.Resource.Identifier = entry.InstanceID.String()
	}

	return l.logger.Enqueue(auditEvent)
}

// Shutdown gracefully shuts down the async audit logger.
func (l *AsyncAuditLogger) Shutdown(timeout time.Duration) error {
	return l.logger.Shutdown(timeout)
}

// ChannelStatus returns the current channel usage statistics.
func (l *AsyncAuditLogger) ChannelStatus() (used, capacity int) {
	return l.logger.ChannelStatus()
}

// ExtractClientIP extracts the client IP from context
func ExtractClientIP(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if ip, ok := ctx.Value(contextKeyClientIP).(string); ok {
		return ip
	}
	return ""
}

// marshalDetails marshals audit details to JSON string
func marshalDetails(details map[string]interface{}) (string, error) {
	if len(details) == 0 {
		return "", nil
	}
	payload := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"details":   details,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to serialize audit details: %w", err)
	}
	return string(data), nil
}
