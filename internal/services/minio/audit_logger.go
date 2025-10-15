package minio

import (
	"context"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	repo "github.com/ysicing/tiga/internal/repository/minio"
)

type AuditLogger struct {
	repo *repo.AuditRepository
}

func NewAuditLogger(r *repo.AuditRepository) *AuditLogger { return &AuditLogger{repo: r} }

func (l *AuditLogger) LogOperation(ctx context.Context, instanceID uuid.UUID, opType, resType, resName, action, status, message string, operatorID *uuid.UUID, operatorName, clientIP string, details models.JSONB) error {
	entry := &models.MinIOAuditLog{
		InstanceID:    instanceID,
		OperationType: opType,
		ResourceType:  resType,
		ResourceName:  resName,
		Action:        action,
		OperatorID:    operatorID,
		OperatorName:  operatorName,
		ClientIP:      clientIP,
		Status:        status,
		ErrorMessage:  message,
		Details:       details,
	}
	return l.repo.Create(ctx, entry)
}
