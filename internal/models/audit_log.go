package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID       uuid.UUID  `gorm:"type:char(36);primary_key" json:"id"`
	UserID   *uuid.UUID `gorm:"type:char(36);index" json:"user_id,omitempty"`
	Username string     `gorm:"type:varchar(64)" json:"username"` // Snapshot, prevents loss if user is deleted

	// Cluster context (Phase 4 enhancement)
	ClusterID   *uint  `gorm:"index" json:"cluster_id,omitempty"`
	ClusterName string `gorm:"type:varchar(255)" json:"cluster_name,omitempty"` // Snapshot

	// Operation
	Action       string     `gorm:"type:varchar(64);not null;index" json:"action"`        // create, update, delete, login, logout
	ResourceType string     `gorm:"type:varchar(64);not null;index" json:"resource_type"` // user, instance, role, etc.
	ResourceID   *uuid.UUID `gorm:"type:char(36);index" json:"resource_id,omitempty"`
	ResourceName string     `gorm:"type:varchar(255)" json:"resource_name,omitempty"`

	// Details
	Description string `gorm:"type:text" json:"description"`
	Changes     JSONB  `gorm:"type:text" json:"changes,omitempty"` // Change details

	// Request information
	IPAddress string `gorm:"type:varchar(45)" json:"ip_address,omitempty"` // Changed from inet to varchar(45) for IPv6
	UserAgent string `gorm:"type:text" json:"user_agent,omitempty"`
	RequestID string `gorm:"type:varchar(128)" json:"request_id,omitempty"` // Correlation ID

	// Result
	Status       string `gorm:"type:varchar(32);not null;index" json:"status"` // success, failure
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`

	// Timestamp
	CreatedAt time.Time `gorm:"index" json:"created_at"`

	// Associations
	User    *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Cluster *Cluster `gorm:"foreignKey:ClusterID" json:"cluster,omitempty"`
}

// TableName overrides the table name
func (AuditLog) TableName() string {
	return "audit_logs"
}

// BeforeCreate hook
func (al *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if al.ID == uuid.Nil {
		al.ID = uuid.New()
	}
	return nil
}

// IsSuccess checks if the operation was successful
func (al *AuditLog) IsSuccess() bool {
	return al.Status == "success"
}

// Docker-specific operation types (Action field values)
const (
	DockerActionContainerStart   = "container_start"
	DockerActionContainerStop    = "container_stop"
	DockerActionContainerRestart = "container_restart"
	DockerActionContainerPause   = "container_pause"
	DockerActionContainerUnpause = "container_unpause"
	DockerActionContainerDelete  = "container_delete"
	DockerActionContainerExec    = "container_exec"
	DockerActionImageDelete      = "image_delete"
	DockerActionImagePull        = "image_pull"
	DockerActionImageTag         = "image_tag"
	DockerActionInstanceCreate   = "instance_create"
	DockerActionInstanceUpdate   = "instance_update"
	DockerActionInstanceDelete   = "instance_delete"
)

// Docker-specific resource types (ResourceType field values)
const (
	DockerResourceTypeContainer = "docker_container"
	DockerResourceTypeImage     = "docker_image"
	DockerResourceTypeInstance  = "docker_instance"
)

// DockerOperationDetails contains Docker-specific operation details stored in Changes field
type DockerOperationDetails struct {
	InstanceID   uuid.UUID              `json:"instance_id"`
	InstanceName string                 `json:"instance_name"`
	AgentID      uuid.UUID              `json:"agent_id,omitempty"`
	StateBefore  string                 `json:"state_before,omitempty"` // Container/Image state before operation
	StateAfter   string                 `json:"state_after,omitempty"`  // Container/Image state after operation
	Success      bool                   `json:"success"`
	Duration     int64                  `json:"duration,omitempty"`   // Operation duration in milliseconds
	ExtraData    map[string]interface{} `json:"extra_data,omitempty"` // Additional operation-specific data
}

// NewDockerAuditLog creates a new audit log entry for Docker operations
func NewDockerAuditLog(userID uuid.UUID, username, action, resourceType string, resourceID uuid.UUID, resourceName string, details DockerOperationDetails) *AuditLog {
	status := "success"
	if !details.Success {
		status = "failure"
	}

	// Convert DockerOperationDetails to JSONB (map[string]interface{})
	changesJSON := make(JSONB)
	detailsBytes, err := json.Marshal(details)
	if err == nil {
		_ = json.Unmarshal(detailsBytes, &changesJSON)
	}

	return &AuditLog{
		ID:           uuid.New(),
		UserID:       &userID,
		Username:     username,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		ResourceName: resourceName,
		Changes:      changesJSON,
		Status:       status,
		CreatedAt:    time.Now(),
	}
}

// ParseDockerDetails parses the Changes field into DockerOperationDetails
func (al *AuditLog) ParseDockerDetails() (*DockerOperationDetails, error) {
	if al.Changes == nil || len(al.Changes) == 0 {
		return nil, nil
	}

	// Convert JSONB (map[string]interface{}) to DockerOperationDetails
	changesBytes, err := json.Marshal(al.Changes)
	if err != nil {
		return nil, err
	}

	var details DockerOperationDetails
	if err := json.Unmarshal(changesBytes, &details); err != nil {
		return nil, err
	}

	return &details, nil
}
