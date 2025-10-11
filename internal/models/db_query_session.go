package models

import (
	"time"

	"github.com/google/uuid"
)

// QuerySession captures the execution metadata for a database query or command run through Tiga.
type QuerySession struct {
	BaseModelWithoutSoftDelete

	InstanceID   uuid.UUID         `gorm:"type:char(36);not null;index:idx_db_query_session_instance" json:"instance_id"`
	Instance     *DatabaseInstance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
	ExecutedBy   string            `gorm:"type:varchar(100);not null;index:idx_db_query_session_user" json:"executed_by"`
	DatabaseName string            `gorm:"type:varchar(100)" json:"database_name"`

	QuerySQL       string     `gorm:"type:text;not null" json:"query_sql"`
	QueryType      string     `gorm:"type:varchar(20);index:idx_db_query_session_type" json:"query_type"`
	Status         string     `gorm:"type:varchar(20);not null;index:idx_db_query_session_status" json:"status"`
	ErrorMessage   string     `gorm:"type:text" json:"error_msg,omitempty"`
	StartedAt      time.Time  `gorm:"not null" json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	DurationMillis int        `gorm:"default:0" json:"duration"`
	RowCount       int        `gorm:"default:0" json:"row_count"`
	BytesReturned  int64      `gorm:"default:0" json:"bytes_returned"`
	ClientIP       string     `gorm:"type:varchar(50)" json:"client_ip"`
}

// TableName overrides the default table name.
func (QuerySession) TableName() string {
	return "db_query_sessions"
}
