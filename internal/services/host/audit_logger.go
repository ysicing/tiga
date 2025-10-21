package host

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/audit"
)

// AuditLogger provides audit logging for host management subsystem
// T038: 迁移到统一 AuditEvent 模型
type AuditLogger struct {
	logger *audit.AsyncLogger[*models.AuditEvent]
}

// NewAuditLogger constructs an AuditLogger for host subsystem
func NewAuditLogger(repo repository.AuditEventRepository, config *audit.Config) *AuditLogger {
	if config == nil {
		config = audit.DefaultConfig()
	}
	return &AuditLogger{
		logger: audit.NewAsyncLogger[*models.AuditEvent](repo, "Host", config),
	}
}

// AuditEntry represents a host audit log entry
type AuditEntry struct {
	HostNodeID  uuid.UUID
	UserID      *uuid.UUID
	Username    string
	Action      string // terminal_created, agent_connected, node_updated, etc.
	ActionType  string // terminal, agent, system, user
	Description string
	Metadata    string // JSON metadata
	ClientIP    string
	UserAgent   string
}

// LogActivity logs a host management activity
func (l *AuditLogger) LogActivity(ctx context.Context, entry AuditEntry) error {
	// Build Data field with host-specific information
	data := map[string]string{
		"action_type": entry.ActionType,
	}

	if entry.Description != "" {
		data["description"] = entry.Description
	}

	if entry.Metadata != "" {
		// Parse metadata JSON and add to data
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(entry.Metadata), &metadata); err == nil {
			for k, v := range metadata {
				if strVal, ok := v.(string); ok {
					data[k] = strVal
				} else {
					jsonBytes, _ := json.Marshal(v)
					data[k] = string(jsonBytes)
				}
			}
		} else {
			// If not valid JSON, store as is
			data["metadata"] = entry.Metadata
		}
	}

	// Map to unified AuditEvent model
	auditEvent := &models.AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now().UnixMilli(),
		Subsystem: models.SubsystemHost,
		Action:    models.Action(entry.Action),
		Resource: models.Resource{
			Type:       models.ResourceTypeHost,
			Identifier: entry.HostNodeID.String(),
			Data: map[string]string{
				"host_node_id": entry.HostNodeID.String(),
			},
		},
		ClientIP:  entry.ClientIP,
		UserAgent: entry.UserAgent,
		Data:      data,
		CreatedAt: time.Now(),
	}

	// Set user information if provided
	if entry.UserID != nil {
		auditEvent.User = models.Principal{
			UID:      entry.UserID.String(),
			Username: entry.Username,
			Type:     models.PrincipalTypeUser,
		}
	} else {
		// System or anonymous action
		auditEvent.User = models.Principal{
			Username: "system",
			Type:     models.PrincipalTypeSystem,
		}
	}

	return l.logger.Enqueue(auditEvent)
}

// Close gracefully shuts down the audit logger
func (l *AuditLogger) Close() error {
	// AsyncLogger doesn't have a Close method, resources are cleaned up automatically
	return nil
}
