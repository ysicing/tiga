package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

func TestK8sRecordingTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		typeVal  string
		expected string
	}{
		{"K8s Node", RecordingTypeK8sNode, "k8s_node"},
		{"K8s Pod", RecordingTypeK8sPod, "k8s_pod"},
		{"Docker", RecordingTypeDocker, "docker"},
		{"WebSSH", RecordingTypeWebSSH, "webssh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.typeVal != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.typeVal)
			}
		})
	}
}

func TestMaxRecordingDuration(t *testing.T) {
	if MaxRecordingDuration != 7200 {
		t.Errorf("Expected MaxRecordingDuration to be 7200, got %d", MaxRecordingDuration)
	}
}

func TestValidateTypeMetadata_K8sNode(t *testing.T) {
	tests := []struct {
		name        string
		recording   *TerminalRecording
		shouldError bool
	}{
		{
			name: "Valid K8s Node metadata",
			recording: &TerminalRecording{
				RecordingType: RecordingTypeK8sNode,
				TypeMetadata: datatypes.JSON([]byte(`{
					"cluster_id": "550e8400-e29b-41d4-a716-446655440000",
					"node_name": "node-1"
				}`)),
			},
			shouldError: false,
		},
		{
			name: "Missing cluster_id",
			recording: &TerminalRecording{
				RecordingType: RecordingTypeK8sNode,
				TypeMetadata: datatypes.JSON([]byte(`{
					"node_name": "node-1"
				}`)),
			},
			shouldError: true,
		},
		{
			name: "Missing node_name",
			recording: &TerminalRecording{
				RecordingType: RecordingTypeK8sNode,
				TypeMetadata: datatypes.JSON([]byte(`{
					"cluster_id": "550e8400-e29b-41d4-a716-446655440000"
				}`)),
			},
			shouldError: true,
		},
		{
			name: "Empty metadata",
			recording: &TerminalRecording{
				RecordingType: RecordingTypeK8sNode,
				TypeMetadata:  datatypes.JSON([]byte(`{}`)),
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.recording.ValidateTypeMetadata()
			if tt.shouldError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateTypeMetadata_K8sPod(t *testing.T) {
	tests := []struct {
		name        string
		recording   *TerminalRecording
		shouldError bool
	}{
		{
			name: "Valid K8s Pod metadata",
			recording: &TerminalRecording{
				RecordingType: RecordingTypeK8sPod,
				TypeMetadata: datatypes.JSON([]byte(`{
					"cluster_id": "550e8400-e29b-41d4-a716-446655440000",
					"namespace": "default",
					"pod_name": "nginx-pod",
					"container_name": "nginx"
				}`)),
			},
			shouldError: false,
		},
		{
			name: "Missing container_name",
			recording: &TerminalRecording{
				RecordingType: RecordingTypeK8sPod,
				TypeMetadata: datatypes.JSON([]byte(`{
					"cluster_id": "550e8400-e29b-41d4-a716-446655440000",
					"namespace": "default",
					"pod_name": "nginx-pod"
				}`)),
			},
			shouldError: true,
		},
		{
			name: "Missing namespace",
			recording: &TerminalRecording{
				RecordingType: RecordingTypeK8sPod,
				TypeMetadata: datatypes.JSON([]byte(`{
					"cluster_id": "550e8400-e29b-41d4-a716-446655440000",
					"pod_name": "nginx-pod",
					"container_name": "nginx"
				}`)),
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.recording.ValidateTypeMetadata()
			if tt.shouldError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateDuration(t *testing.T) {
	tests := []struct {
		name        string
		duration    int
		shouldError bool
	}{
		{"Valid duration - 1 hour", 3600, false},
		{"Valid duration - exactly 2 hours", 7200, false},
		{"Invalid duration - over 2 hours", 7201, true},
		{"Invalid duration - 3 hours", 10800, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recording := &TerminalRecording{Duration: tt.duration}
			err := recording.ValidateDuration()
			if tt.shouldError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestIsExpiredMethod(t *testing.T) {
	now := time.Now()
	expiredTime := now.AddDate(0, 0, -91) // 91 days ago
	recentTime := now.AddDate(0, 0, -89)  // 89 days ago

	tests := []struct {
		name        string
		recording   *TerminalRecording
		shouldExpire bool
	}{
		{
			name: "Expired recording (91 days old)",
			recording: &TerminalRecording{
				EndedAt: &expiredTime,
			},
			shouldExpire: true,
		},
		{
			name: "Not expired recording (89 days old)",
			recording: &TerminalRecording{
				EndedAt: &recentTime,
			},
			shouldExpire: false,
		},
		{
			name: "Active recording (no EndedAt)",
			recording: &TerminalRecording{
				EndedAt: nil,
			},
			shouldExpire: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isExpired := tt.recording.IsExpired()
			if isExpired != tt.shouldExpire {
				t.Errorf("Expected IsExpired()=%v, got %v", tt.shouldExpire, isExpired)
			}
		})
	}
}

func TestK8sTypeMetadataStructure(t *testing.T) {
	// Test creating K8s node recording
	nodeMetadata := map[string]string{
		"cluster_id": uuid.New().String(),
		"node_name":  "test-node",
	}
	nodeJSON, _ := json.Marshal(nodeMetadata)
	nodeRecording := &TerminalRecording{
		RecordingType: RecordingTypeK8sNode,
		TypeMetadata:  datatypes.JSON(nodeJSON),
	}

	if err := nodeRecording.ValidateTypeMetadata(); err != nil {
		t.Errorf("Valid K8s node metadata failed validation: %v", err)
	}

	// Test creating K8s pod recording
	podMetadata := map[string]string{
		"cluster_id":     uuid.New().String(),
		"namespace":      "default",
		"pod_name":       "test-pod",
		"container_name": "test-container",
	}
	podJSON, _ := json.Marshal(podMetadata)
	podRecording := &TerminalRecording{
		RecordingType: RecordingTypeK8sPod,
		TypeMetadata:  datatypes.JSON(podJSON),
	}

	if err := podRecording.ValidateTypeMetadata(); err != nil {
		t.Errorf("Valid K8s pod metadata failed validation: %v", err)
	}
}
