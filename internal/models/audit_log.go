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
	// Instance operations
	DockerActionListInstances  = "list_instances"
	DockerActionGetInstance    = "get_instance"
	DockerActionCreateInstance = "create_instance"
	DockerActionUpdateInstance = "update_instance"
	DockerActionDeleteInstance = "delete_instance"
	DockerActionTestConnection = "test_connection"

	// Container operations
	DockerActionListContainers    = "list_containers"
	DockerActionGetContainer      = "get_container"
	DockerActionStartContainer    = "start_container"
	DockerActionStopContainer     = "stop_container"
	DockerActionRestartContainer  = "restart_container"
	DockerActionPauseContainer    = "pause_container"
	DockerActionUnpauseContainer  = "unpause_container"
	DockerActionDeleteContainer   = "delete_container"
	DockerActionExecContainer     = "exec_container"
	DockerActionGetContainerLogs  = "get_container_logs"
	DockerActionGetContainerStats = "get_container_stats"

	// Image operations
	DockerActionListImages  = "list_images"
	DockerActionGetImage    = "get_image"
	DockerActionPullImage   = "pull_image"
	DockerActionDeleteImage = "delete_image"
	DockerActionTagImage    = "tag_image"

	// Network operations
	DockerActionListNetworks      = "list_networks"
	DockerActionGetNetwork        = "get_network"
	DockerActionCreateNetwork     = "create_network"
	DockerActionDeleteNetwork     = "delete_network"
	DockerActionConnectNetwork    = "connect_network"
	DockerActionDisconnectNetwork = "disconnect_network"

	// Volume operations
	DockerActionListVolumes  = "list_volumes"
	DockerActionGetVolume    = "get_volume"
	DockerActionCreateVolume = "create_volume"
	DockerActionDeleteVolume = "delete_volume"
	DockerActionPruneVolumes = "prune_volumes"

	// System operations
	DockerActionGetSystemInfo = "get_system_info"
	DockerActionGetVersion    = "get_version"
	DockerActionGetDiskUsage  = "get_disk_usage"
	DockerActionPing          = "ping"
	DockerActionGetEvents     = "get_events"

	// Terminal recording operations
	DockerActionListRecordings  = "list_recordings"
	DockerActionGetRecording    = "get_recording"
	DockerActionDeleteRecording = "delete_recording"
	DockerActionPlayRecording   = "play_recording"
)

// Docker-specific resource types (ResourceType field values)
const (
	DockerResourceTypeInstance  = "docker_instance"
	DockerResourceTypeContainer = "docker_container"
	DockerResourceTypeImage     = "docker_image"
	DockerResourceTypeNetwork   = "docker_network"
	DockerResourceTypeVolume    = "docker_volume"
	DockerResourceTypeSystem    = "docker_system"
	DockerResourceTypeRecording = "docker_recording"
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
