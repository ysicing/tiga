package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	dbrepo "github.com/ysicing/tiga/internal/repository/database"
)

type contextKey string

const contextKeyClientIP contextKey = "client_ip"

// AuditLogger records database management actions.
type AuditLogger struct {
	repo *dbrepo.AuditLogRepository
}

// NewAuditLogger constructs an AuditLogger.
func NewAuditLogger(repo *dbrepo.AuditLogRepository) *AuditLogger {
	return &AuditLogger{repo: repo}
}

// AuditEntry describes an action for auditing.
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

// LogAction persists an audit entry.
func (l *AuditLogger) LogAction(ctx context.Context, entry AuditEntry) error {
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

	details, err := marshalDetails(entry.Details)
	if err != nil {
		return err
	}

	log := &models.DatabaseAuditLog{
		Operator:   entry.Operator,
		Action:     entry.Action,
		TargetType: entry.TargetType,
		TargetName: entry.TargetName,
		Details:    details,
		Success:    entry.Success,
		ClientIP:   clientIP,
	}
	if entry.InstanceID != nil {
		log.InstanceID = entry.InstanceID
	}
	if entry.Error != nil {
		log.ErrorMessage = entry.Error.Error()
	}

	return l.repo.Create(ctx, log)
}

// ExtractClientIP retrieves the client IP from context if provided.
func ExtractClientIP(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if ip, ok := ctx.Value(contextKeyClientIP).(string); ok {
		return ip
	}
	return ""
}

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
