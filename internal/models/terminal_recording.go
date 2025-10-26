package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TerminalRecording represents a terminal session recording for audit and playback
// Supports multiple terminal types: Docker, WebSSH, K8s Node, K8s Pod
type TerminalRecording struct {
	BaseModel
	SessionID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"session_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Username  string    `gorm:"type:varchar(255);not null" json:"username"`

	// Recording type (unified support for multiple terminal types)
	RecordingType string         `gorm:"type:varchar(50);not null;default:'docker';index" json:"recording_type"` // docker, webssh, k8s_node, k8s_pod
	TypeMetadata  datatypes.JSON `gorm:"type:jsonb" json:"type_metadata"`                                        // Type-specific metadata (instance_id, container_id, host_id, etc.)

	// Legacy Docker fields (kept for backward compatibility with existing code)
	InstanceID  uuid.UUID `gorm:"type:uuid;index" json:"instance_id,omitempty"`
	ContainerID string    `gorm:"type:varchar(255)" json:"container_id,omitempty"`

	// Recording metadata
	StartedAt time.Time  `gorm:"not null;index" json:"started_at"`
	EndedAt   *time.Time `gorm:"index" json:"ended_at,omitempty"`
	Duration  int        `gorm:"default:0;index" json:"duration"` // Duration in seconds

	// Storage information
	StorageType string `gorm:"type:varchar(50);default:'local';index" json:"storage_type"` // local, minio
	StoragePath string `gorm:"type:text;not null" json:"storage_path"`                     // Path in MinIO or local filesystem
	FileSize    int64  `gorm:"default:0;index" json:"file_size"`                           // File size in bytes
	Format      string `gorm:"type:varchar(50);default:'asciinema'" json:"format"`         // Recording format

	// Terminal configuration
	Rows  int    `gorm:"default:30" json:"rows"`
	Cols  int    `gorm:"default:120" json:"cols"`
	Shell string `gorm:"type:varchar(255)" json:"shell"`

	// Additional metadata
	ClientIP    string `gorm:"type:varchar(255)" json:"client_ip"`
	Description string `gorm:"type:text" json:"description,omitempty"`
	Tags        string `gorm:"type:text" json:"tags,omitempty"` // Comma-separated tags

	// Relations
	Instance *DockerInstance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
}

// TableName specifies the table name for TerminalRecording
func (TerminalRecording) TableName() string {
	return "terminal_recordings"
}

// AfterMigrate creates composite indexes for performance optimization
func (TerminalRecording) AfterMigrate(tx *gorm.DB) error {
	// Create composite cleanup index for efficient cleanup queries
	// This index is used by FindExpired and FindInvalid repository methods
	dbDialect := tx.Dialector.Name()

	var sql string
	switch dbDialect {
	case "postgres", "sqlite":
		// PostgreSQL and SQLite support partial indexes with WHERE clause
		sql = `CREATE INDEX IF NOT EXISTS idx_terminal_recordings_cleanup
		       ON terminal_recordings(ended_at, file_size, duration)
		       WHERE ended_at IS NOT NULL`
	case "mysql":
		// MySQL doesn't support partial indexes, create full composite index
		sql = `CREATE INDEX idx_terminal_recordings_cleanup
		       ON terminal_recordings(ended_at, file_size, duration)`
	default:
		// Skip index creation for unsupported databases
		return nil
	}

	// For MySQL, ignore "Duplicate key name" errors (index already exists)
	if err := tx.Exec(sql).Error; err != nil && dbDialect == "mysql" {
		// Log but don't fail - index may already exist
		return nil
	} else if err != nil {
		return err
	}

	return nil
}

// RecordingFrame represents a single frame in the terminal recording (asciinema format)
type RecordingFrame struct {
	Timestamp float64 `json:"timestamp"` // Relative timestamp in seconds from start
	Type      string  `json:"type"`      // "o" for output, "i" for input
	Data      string  `json:"data"`      // Terminal data
}

// AsciinemaHeader represents the header of an asciinema recording file (v2 format)
type AsciinemaHeader struct {
	Version   int                    `json:"version"`   // Always 2
	Width     int                    `json:"width"`     // Terminal width
	Height    int                    `json:"height"`    // Terminal height
	Timestamp int64                  `json:"timestamp"` // Unix timestamp
	Title     string                 `json:"title,omitempty"`
	Env       map[string]string      `json:"env,omitempty"`
	Theme     map[string]interface{} `json:"theme,omitempty"`
}

// RecordingMetadata contains metadata about a recording for API responses
type RecordingMetadata struct {
	ID            uuid.UUID       `json:"id"`
	SessionID     uuid.UUID       `json:"session_id"`
	Username      string          `json:"username"`
	RecordingType string          `json:"recording_type"`
	TypeMetadata  datatypes.JSON  `json:"type_metadata,omitempty"`
	StorageType   string          `json:"storage_type"`
	StoragePath   string          `json:"storage_path"`
	FileSize      int64           `json:"file_size"`
	Format        string          `json:"format"`
	Rows          int             `json:"rows"`
	Cols          int             `json:"cols"`
	Shell         string          `json:"shell"`
	StartedAt     time.Time       `json:"started_at"`
	EndedAt       *time.Time      `json:"ended_at,omitempty"`
	Duration      int             `json:"duration"`
	ClientIP      string          `json:"client_ip"`
	Description   string          `json:"description,omitempty"`
	Tags          string          `json:"tags,omitempty"`

	// Legacy fields for backward compatibility
	InstanceID    uuid.UUID       `json:"instance_id,omitempty"`
	ContainerID   string          `json:"container_id,omitempty"`
}

// ToMetadata converts a TerminalRecording to RecordingMetadata
func (r *TerminalRecording) ToMetadata() *RecordingMetadata {
	return &RecordingMetadata{
		ID:            r.ID,
		SessionID:     r.SessionID,
		Username:      r.Username,
		RecordingType: r.RecordingType,
		TypeMetadata:  r.TypeMetadata,
		StorageType:   r.StorageType,
		StoragePath:   r.StoragePath,
		FileSize:      r.FileSize,
		Format:        r.Format,
		Rows:          r.Rows,
		Cols:          r.Cols,
		Shell:         r.Shell,
		StartedAt:     r.StartedAt,
		EndedAt:       r.EndedAt,
		Duration:      r.Duration,
		ClientIP:      r.ClientIP,
		Description:   r.Description,
		Tags:          r.Tags,
		InstanceID:    r.InstanceID,
		ContainerID:   r.ContainerID,
	}
}

// IsCompleted checks if the recording has been finalized
func (r *TerminalRecording) IsCompleted() bool {
	return r.EndedAt != nil
}

// IsExpired checks if the recording has expired based on retention days
func (r *TerminalRecording) IsExpired(retentionDays int) bool {
	if r.EndedAt == nil {
		return false // Active recordings are not expired
	}
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	return r.EndedAt.Before(cutoffTime)
}

// IsInvalid checks if the recording is invalid (zero file size or zero duration)
func (r *TerminalRecording) IsInvalid() bool {
	// Recording is invalid if it has ended but has no content
	if r.EndedAt != nil && (r.FileSize == 0 || r.Duration == 0) {
		return true
	}
	return false
}

// GetDurationString returns human-readable duration
func (r *TerminalRecording) GetDurationString() string {
	if r.Duration < 60 {
		return fmt.Sprintf("%ds", r.Duration)
	}
	minutes := r.Duration / 60
	seconds := r.Duration % 60
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}

// GetFileSizeString returns human-readable file size
func (r *TerminalRecording) GetFileSizeString() string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	size := float64(r.FileSize)
	switch {
	case r.FileSize >= GB:
		return fmt.Sprintf("%.2f GB", size/GB)
	case r.FileSize >= MB:
		return fmt.Sprintf("%.2f MB", size/MB)
	case r.FileSize >= KB:
		return fmt.Sprintf("%.2f KB", size/KB)
	default:
		return fmt.Sprintf("%d B", r.FileSize)
	}
}
