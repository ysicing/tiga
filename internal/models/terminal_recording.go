package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Recording type constants
const (
	RecordingTypeDocker  = "docker"
	RecordingTypeWebSSH  = "webssh"
	RecordingTypeK8sNode = "k8s_node"
	RecordingTypeK8sPod  = "k8s_pod"
)

// MaxRecordingDuration is the maximum recording duration (2 hours in seconds)
const MaxRecordingDuration = 7200

// TerminalRecording represents a terminal session recording for audit and playback
// Supports multiple terminal types: Docker, WebSSH, K8s Node, K8s Pod
type TerminalRecording struct {
	BaseModel
	SessionID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"session_id"`
	// UserID is optional for K8s terminal recordings (can be NULL to avoid foreign key constraint)
	UserID   *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	Username string     `gorm:"type:varchar(255);not null" json:"username"`

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

	var sqls []string

	// Index 1: Cleanup index (existing)
	switch dbDialect {
	case "postgres", "sqlite":
		// PostgreSQL and SQLite support partial indexes with WHERE clause
		sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_terminal_recordings_cleanup
		       ON terminal_recordings(ended_at, file_size, duration)
		       WHERE ended_at IS NOT NULL`)
	case "mysql":
		// MySQL doesn't support partial indexes, create full composite index
		sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_terminal_recordings_cleanup
		       ON terminal_recordings(ended_at, file_size, duration)`)
	}

	// Index 2: K8s cluster filtering (010-k8s-pod-009 T003)
	// Enables fast queries by cluster_id from type_metadata JSONB field
	switch dbDialect {
	case "postgres":
		sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_terminal_recordings_k8s_cluster
		       ON terminal_recordings((type_metadata->>'cluster_id'))
		       WHERE recording_type IN ('k8s_node', 'k8s_pod')`)
	case "sqlite":
		// SQLite requires JSON_EXTRACT syntax
		sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_terminal_recordings_k8s_cluster
		       ON terminal_recordings(json_extract(type_metadata, '$.cluster_id'))
		       WHERE recording_type IN ('k8s_node', 'k8s_pod')`)
	case "mysql":
		// MySQL requires virtual column for JSON indexing
		sqls = append(sqls, `ALTER TABLE terminal_recordings
		       ADD COLUMN IF NOT EXISTS k8s_cluster_id VARCHAR(255)
		       AS (JSON_UNQUOTE(JSON_EXTRACT(type_metadata, '$.cluster_id'))) VIRTUAL`)
		sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_terminal_recordings_k8s_cluster
		       ON terminal_recordings(k8s_cluster_id)
		       WHERE recording_type IN ('k8s_node', 'k8s_pod')`)
	}

	// Index 3: K8s node filtering (010-k8s-pod-009 T003)
	switch dbDialect {
	case "postgres":
		sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_terminal_recordings_k8s_node
		       ON terminal_recordings((type_metadata->>'node_name'))
		       WHERE recording_type = 'k8s_node'`)
	case "sqlite":
		sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_terminal_recordings_k8s_node
		       ON terminal_recordings(json_extract(type_metadata, '$.node_name'))
		       WHERE recording_type = 'k8s_node'`)
	case "mysql":
		sqls = append(sqls, `ALTER TABLE terminal_recordings
		       ADD COLUMN IF NOT EXISTS k8s_node_name VARCHAR(255)
		       AS (JSON_UNQUOTE(JSON_EXTRACT(type_metadata, '$.node_name'))) VIRTUAL`)
		sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_terminal_recordings_k8s_node
		       ON terminal_recordings(k8s_node_name)
		       WHERE recording_type = 'k8s_node'`)
	}

	// Index 4: K8s Pod filtering (010-k8s-pod-009 T003)
	switch dbDialect {
	case "postgres":
		sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_terminal_recordings_k8s_pod
		       ON terminal_recordings((type_metadata->>'pod_name'))
		       WHERE recording_type = 'k8s_pod'`)
	case "sqlite":
		sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_terminal_recordings_k8s_pod
		       ON terminal_recordings(json_extract(type_metadata, '$.pod_name'))
		       WHERE recording_type = 'k8s_pod'`)
	case "mysql":
		sqls = append(sqls, `ALTER TABLE terminal_recordings
		       ADD COLUMN IF NOT EXISTS k8s_pod_name VARCHAR(255)
		       AS (JSON_UNQUOTE(JSON_EXTRACT(type_metadata, '$.pod_name'))) VIRTUAL`)
		sqls = append(sqls, `CREATE INDEX IF NOT EXISTS idx_terminal_recordings_k8s_pod
		       ON terminal_recordings(k8s_pod_name)
		       WHERE recording_type = 'k8s_pod'`)
	}

	// Execute all SQL statements
	for _, sql := range sqls {
		if sql == "" {
			continue
		}
		if err := tx.Exec(sql).Error; err != nil {
			// For MySQL, ignore "Duplicate" errors (index/column already exists)
			if dbDialect == "mysql" {
				// Log but don't fail
				continue
			}
			// For other databases, only fail on non-"already exists" errors
			if !isAlreadyExistsError(err) {
				return err
			}
		}
	}

	return nil
}

// isAlreadyExistsError checks if the error is an "already exists" error
func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	// PostgreSQL: "already exists"
	// SQLite: "already exists", "duplicate column name"
	// MySQL: "Duplicate", "already exists"
	return strings.Contains(errMsg, "already exists") ||
		strings.Contains(errMsg, "Duplicate") ||
		strings.Contains(errMsg, "duplicate")
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

// ValidateTypeMetadata validates the type_metadata field based on recording_type
func (r *TerminalRecording) ValidateTypeMetadata() error {
	if len(r.TypeMetadata) == 0 {
		return nil // Empty metadata is allowed for legacy recordings
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(r.TypeMetadata, &metadata); err != nil {
		return fmt.Errorf("invalid type_metadata JSON: %w", err)
	}

	switch r.RecordingType {
	case RecordingTypeK8sNode:
		if clusterID, ok := metadata["cluster_id"].(string); !ok || clusterID == "" {
			return errors.New("k8s_node recording requires cluster_id in type_metadata")
		}
		if nodeName, ok := metadata["node_name"].(string); !ok || nodeName == "" {
			return errors.New("k8s_node recording requires node_name in type_metadata")
		}
	case RecordingTypeK8sPod:
		requiredFields := []string{"cluster_id", "namespace", "pod_name", "container_name"}
		for _, field := range requiredFields {
			if val, ok := metadata[field].(string); !ok || val == "" {
				return fmt.Errorf("k8s_pod recording requires %s in type_metadata", field)
			}
		}
	}
	return nil
}

// ValidateDuration checks if recording duration exceeds 2-hour limit
func (r *TerminalRecording) ValidateDuration() error {
	if r.Duration > MaxRecordingDuration {
		return fmt.Errorf("recording duration %d seconds exceeds maximum %d seconds (2 hours)", r.Duration, MaxRecordingDuration)
	}
	return nil
}
