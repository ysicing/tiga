package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TerminalRecording represents a terminal session recording for audit and playback
type TerminalRecording struct {
	BaseModel
	SessionID   uuid.UUID `gorm:"type:uuid;not null;index" json:"session_id"`
	InstanceID  uuid.UUID `gorm:"type:uuid;not null;index" json:"instance_id"`
	ContainerID string    `gorm:"type:varchar(255);not null" json:"container_id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Username    string    `gorm:"type:varchar(255);not null" json:"username"`

	// Recording metadata
	StartedAt time.Time  `gorm:"not null" json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Duration  int        `gorm:"default:0" json:"duration"` // Duration in seconds

	// Storage information
	StoragePath string `gorm:"type:text;not null" json:"storage_path"`             // Path in MinIO or local filesystem
	FileSize    int64  `gorm:"default:0" json:"file_size"`                         // File size in bytes
	Format      string `gorm:"type:varchar(50);default:'asciinema'" json:"format"` // Recording format

	// Terminal configuration
	Rows  int    `gorm:"default:30" json:"rows"`
	Cols  int    `gorm:"default:120" json:"cols"`
	Shell string `gorm:"type:varchar(255)" json:"shell"`

	// Additional metadata
	ClientIP    string `gorm:"type:varchar(255)" json:"client_ip"`
	Description string `gorm:"type:text" json:"description,omitempty"`

	// Relations
	Instance *DockerInstance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
}

// TableName specifies the table name for TerminalRecording
func (TerminalRecording) TableName() string {
	return "terminal_recordings"
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
	ID          uuid.UUID  `json:"id"`
	SessionID   uuid.UUID  `json:"session_id"`
	InstanceID  uuid.UUID  `json:"instance_id"`
	ContainerID string     `json:"container_id"`
	Username    string     `json:"username"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	Duration    int        `json:"duration"`
	FileSize    int64      `json:"file_size"`
	ClientIP    string     `json:"client_ip"`
	Description string     `json:"description,omitempty"`
}

// ToMetadata converts a TerminalRecording to RecordingMetadata
func (r *TerminalRecording) ToMetadata() *RecordingMetadata {
	return &RecordingMetadata{
		ID:          r.ID,
		SessionID:   r.SessionID,
		InstanceID:  r.InstanceID,
		ContainerID: r.ContainerID,
		Username:    r.Username,
		StartedAt:   r.StartedAt,
		EndedAt:     r.EndedAt,
		Duration:    r.Duration,
		FileSize:    r.FileSize,
		ClientIP:    r.ClientIP,
		Description: r.Description,
	}
}

// IsCompleted checks if the recording has been finalized
func (r *TerminalRecording) IsCompleted() bool {
	return r.EndedAt != nil
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
