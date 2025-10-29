package models

import (
	"testing"
	"time"
)

func TestK8sActionConstants(t *testing.T) {
	tests := []struct {
		name     string
		action   Action
		expected string
	}{
		{"Create Resource", ActionCreateResource, "CreateResource"},
		{"Update Resource", ActionUpdateResource, "UpdateResource"},
		{"Delete Resource", ActionDeleteResource, "DeleteResource"},
		{"View Resource", ActionViewResource, "ViewResource"},
		{"Node Terminal Access", ActionNodeTerminalAccess, "NodeTerminalAccess"},
		{"Pod Terminal Access", ActionPodTerminalAccess, "PodTerminalAccess"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.action) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.action))
			}
			// Verify action validates correctly
			if err := tt.action.Validate(); err != nil {
				t.Errorf("Expected valid action, got error: %v", err)
			}
		})
	}
}

func TestK8sResourceTypeConstants(t *testing.T) {
	tests := []struct {
		name         string
		resourceType ResourceType
		expected     string
	}{
		{"K8s Node", ResourceTypeK8sNode, "k8s_node"},
		{"K8s Pod", ResourceTypeK8sPod, "k8s_pod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.resourceType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.resourceType))
			}
			// Verify resource type validates correctly
			if err := tt.resourceType.Validate(); err != nil {
				t.Errorf("Expected valid resource type, got error: %v", err)
			}
		})
	}
}

func TestIsReadOnlyOperation(t *testing.T) {
	tests := []struct {
		name       string
		action     Action
		isReadOnly bool
	}{
		{"ViewResource is read-only", ActionViewResource, true},
		{"ActionRead is read-only", ActionRead, true},
		{"CreateResource is not read-only", ActionCreateResource, false},
		{"UpdateResource is not read-only", ActionUpdateResource, false},
		{"DeleteResource is not read-only", ActionDeleteResource, false},
		{"NodeTerminalAccess is not read-only", ActionNodeTerminalAccess, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ae := &AuditEvent{Action: tt.action}
			result := ae.IsReadOnlyOperation()
			if result != tt.isReadOnly {
				t.Errorf("Expected IsReadOnlyOperation()=%v for action %s, got %v",
					tt.isReadOnly, tt.action, result)
			}
		})
	}
}

func TestIsModifyOperation(t *testing.T) {
	tests := []struct {
		name     string
		action   Action
		isModify bool
	}{
		{"CreateResource is modify", ActionCreateResource, true},
		{"UpdateResource is modify", ActionUpdateResource, true},
		{"DeleteResource is modify", ActionDeleteResource, true},
		{"ActionCreated is modify", ActionCreated, true},
		{"ActionUpdated is modify", ActionUpdated, true},
		{"ActionDeleted is modify", ActionDeleted, true},
		{"ViewResource is not modify", ActionViewResource, false},
		{"ActionRead is not modify", ActionRead, false},
		{"NodeTerminalAccess is not modify", ActionNodeTerminalAccess, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ae := &AuditEvent{Action: tt.action}
			result := ae.IsModifyOperation()
			if result != tt.isModify {
				t.Errorf("Expected IsModifyOperation()=%v for action %s, got %v",
					tt.isModify, tt.action, result)
			}
		})
	}
}

func TestAuditEventIsExpired(t *testing.T) {
	now := time.Now()
	expiredTime := now.AddDate(0, 0, -91) // 91 days ago
	recentTime := now.AddDate(0, 0, -89)  // 89 days ago

	tests := []struct {
		name          string
		createdAt     time.Time
		retentionDays int
		shouldExpire  bool
	}{
		{
			name:          "Expired audit event (91 days old, 90 day retention)",
			createdAt:     expiredTime,
			retentionDays: 90,
			shouldExpire:  true,
		},
		{
			name:          "Not expired audit event (89 days old, 90 day retention)",
			createdAt:     recentTime,
			retentionDays: 90,
			shouldExpire:  false,
		},
		{
			name:          "No retention policy (0 days)",
			createdAt:     expiredTime,
			retentionDays: 0,
			shouldExpire:  false,
		},
		{
			name:          "Negative retention days",
			createdAt:     expiredTime,
			retentionDays: -1,
			shouldExpire:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ae := &AuditEvent{CreatedAt: tt.createdAt}
			isExpired := ae.IsExpired(tt.retentionDays)
			if isExpired != tt.shouldExpire {
				t.Errorf("Expected IsExpired()=%v, got %v", tt.shouldExpire, isExpired)
			}
		})
	}
}

func TestK8sSubsystemValidation(t *testing.T) {
	// Verify SubsystemKubernetes is already defined and valid
	if SubsystemKubernetes != "kubernetes" {
		t.Errorf("Expected SubsystemKubernetes to be 'kubernetes', got %s", SubsystemKubernetes)
	}

	if err := SubsystemKubernetes.Validate(); err != nil {
		t.Errorf("Expected SubsystemKubernetes to be valid, got error: %v", err)
	}
}

func TestK8sAuditEventStructure(t *testing.T) {
	// Test creating a K8s resource creation audit event
	createEvent := &AuditEvent{
		ID:           "test-id-1",
		Timestamp:    time.Now().UnixMilli(),
		Action:       ActionCreateResource,
		ResourceType: ResourceTypeDeployment,
		Resource: Resource{
			Type:       ResourceTypeDeployment,
			Identifier: "nginx-deployment",
			Data: map[string]string{
				"cluster_id":   "cluster-uuid",
				"namespace":    "default",
				"resourceName": "nginx-deployment",
				"success":      "true",
			},
		},
		Subsystem: SubsystemKubernetes,
		User: Principal{
			UID:      "user-uuid",
			Username: "admin",
			Type:     PrincipalTypeUser,
		},
		ClientIP:  "192.168.1.100",
		CreatedAt: time.Now(),
	}

	if err := createEvent.Validate(); err != nil {
		t.Errorf("Valid K8s resource creation event failed validation: %v", err)
	}

	// Test creating a K8s terminal access audit event
	terminalEvent := &AuditEvent{
		ID:           "test-id-2",
		Timestamp:    time.Now().UnixMilli(),
		Action:       ActionNodeTerminalAccess,
		ResourceType: ResourceTypeK8sNode,
		Resource: Resource{
			Type:       ResourceTypeK8sNode,
			Identifier: "node-1",
			Data: map[string]string{
				"cluster_id":   "cluster-uuid",
				"resource_name": "node-1",
				"recording_id": "recording-uuid",
				"success":      "true",
			},
		},
		Subsystem: SubsystemKubernetes,
		User: Principal{
			UID:      "user-uuid",
			Username: "admin",
			Type:     PrincipalTypeUser,
		},
		ClientIP:  "192.168.1.100",
		CreatedAt: time.Now(),
	}

	if err := terminalEvent.Validate(); err != nil {
		t.Errorf("Valid K8s terminal access event failed validation: %v", err)
	}

	// Test creating a K8s Pod terminal access audit event
	podTerminalEvent := &AuditEvent{
		ID:           "test-id-3",
		Timestamp:    time.Now().UnixMilli(),
		Action:       ActionPodTerminalAccess,
		ResourceType: ResourceTypeK8sPod,
		Resource: Resource{
			Type:       ResourceTypeK8sPod,
			Identifier: "nginx-pod",
			Data: map[string]string{
				"cluster_id":     "cluster-uuid",
				"namespace":      "default",
				"pod_name":       "nginx-pod",
				"container_name": "nginx",
				"recording_id":   "recording-uuid",
				"success":        "true",
			},
		},
		Subsystem: SubsystemKubernetes,
		User: Principal{
			UID:      "user-uuid",
			Username: "admin",
			Type:     PrincipalTypeUser,
		},
		ClientIP:  "192.168.1.100",
		CreatedAt: time.Now(),
	}

	if err := podTerminalEvent.Validate(); err != nil {
		t.Errorf("Valid K8s Pod terminal access event failed validation: %v", err)
	}

	// Test read-only operation audit
	viewEvent := &AuditEvent{
		ID:           "test-id-4",
		Timestamp:    time.Now().UnixMilli(),
		Action:       ActionViewResource,
		ResourceType: ResourceTypePod,
		Resource: Resource{
			Type:       ResourceTypePod,
			Identifier: "nginx-pod",
			Data: map[string]string{
				"cluster_id":    "cluster-uuid",
				"namespace":     "default",
				"resource_name": "nginx-pod",
				"success":       "true",
			},
		},
		Subsystem: SubsystemKubernetes,
		User: Principal{
			UID:      "user-uuid",
			Username: "developer",
			Type:     PrincipalTypeUser,
		},
		ClientIP:  "192.168.1.105",
		CreatedAt: time.Now(),
	}

	if err := viewEvent.Validate(); err != nil {
		t.Errorf("Valid K8s view operation event failed validation: %v", err)
	}

	// Verify operation classification
	if !viewEvent.IsReadOnlyOperation() {
		t.Error("ViewResource should be classified as read-only")
	}
	if createEvent.IsReadOnlyOperation() {
		t.Error("CreateResource should not be classified as read-only")
	}
	if !createEvent.IsModifyOperation() {
		t.Error("CreateResource should be classified as modify operation")
	}
}
