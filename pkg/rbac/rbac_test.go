package rbac

import (
	"testing"

	"github.com/ysicing/tiga/internal/models"
)

func TestCanAccess(t *testing.T) {
	// NOTE: RBAC system has been simplified to use IsAdmin flag instead of complex roles
	// Admin users (IsAdmin=true) have full access
	// Regular users (IsAdmin=false) have read-only access (get, list, log)

	tests := []struct {
		name      string
		user      models.User
		resource  string
		verb      string
		cluster   string
		namespace string
		expected  bool
	}{
		{
			name:      "user with no permissions (regular user, non-read verb)",
			user:      models.User{Username: "unprivileged-user", IsAdmin: false},
			resource:  "pod",
			verb:      "delete",
			cluster:   "dev-cluster",
			namespace: "default",
			expected:  false,
		},
		{
			name:      "admin user can access anything",
			user:      models.User{Username: "admin-user", IsAdmin: true},
			resource:  "any-resource",
			verb:      "any-verb",
			cluster:   "any-cluster",
			namespace: "any-namespace",
			expected:  true,
		},
		{
			name:      "viewer can only read",
			user:      models.User{Username: "viewer-user", IsAdmin: false},
			resource:  "pod",
			verb:      "get",
			cluster:   "any-cluster",
			namespace: "default",
			expected:  true,
		},
		{
			name:      "viewer cannot write",
			user:      models.User{Username: "viewer-user", IsAdmin: false},
			resource:  "pod",
			verb:      "create",
			cluster:   "any-cluster",
			namespace: "default",
			expected:  false,
		},
		{
			name:      "developer in correct cluster/namespace/resource (regular user can read)",
			user:      models.User{Username: "dev-user", IsAdmin: false},
			resource:  "pod",
			verb:      "get",
			cluster:   "dev-cluster",
			namespace: "dev",
			expected:  true,
		},
		{
			name:      "developer in wrong cluster (can still read as regular user)",
			user:      models.User{Username: "dev-user", IsAdmin: false},
			resource:  "pod",
			verb:      "create",
			cluster:   "prod-cluster",
			namespace: "dev",
			expected:  false,
		},
		{
			name:      "user with multiple roles (regular user can read)",
			user:      models.User{Username: "multi-role-user", IsAdmin: false},
			resource:  "service",
			verb:      "list",
			cluster:   "prod-cluster",
			namespace: "prod",
			expected:  true,
		},
		{
			name:      "user with OIDC group permissions (regular user can read)",
			user:      models.User{Username: "oidc-user", IsAdmin: false, OIDCGroups: []string{"dev-team"}},
			resource:  "deployment",
			verb:      "get",
			cluster:   "dev-cluster",
			namespace: "dev",
			expected:  true,
		},
		{
			name:      "wildcard in user list (regular user can read)",
			user:      models.User{Username: "any-user", IsAdmin: false},
			resource:  "pod",
			verb:      "get",
			cluster:   "any-cluster",
			namespace: "any-namespace",
			expected:  true,
		},
		{
			name:      "allow all-namespace but not kube-system: access (regular user can read)",
			user:      models.User{Username: "any-user", IsAdmin: false},
			resource:  "pod",
			verb:      "get",
			cluster:   "any-cluster",
			namespace: "any-namespace",
			expected:  true,
		},
		{
			name:      "allow all-namespace but not kube-system: not access (simplified RBAC doesn't enforce namespace restrictions)",
			user:      models.User{Username: "any-user", IsAdmin: false},
			resource:  "pod",
			verb:      "get",
			cluster:   "any-cluster",
			namespace: "kube-system",
			expected:  true, // Simplified RBAC allows read access to all namespaces for regular users
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := CanAccess(tc.user, tc.resource, tc.verb, tc.cluster, tc.namespace)

			if result != tc.expected {
				t.Errorf("Expected CanAccess to return %v but got %v", tc.expected, result)
			}
		})
	}
}

func TestUserHasRole(t *testing.T) {
	tests := []struct {
		name     string
		user     models.User
		roleName string
		expected bool
	}{
		{
			name:     "admin user has admin role",
			user:     models.User{Username: "admin", IsAdmin: true},
			roleName: "admin",
			expected: true,
		},
		{
			name:     "regular user does not have admin role",
			user:     models.User{Username: "user", IsAdmin: false},
			roleName: "admin",
			expected: false,
		},
		{
			name:     "any role other than admin always returns false",
			user:     models.User{Username: "user", IsAdmin: false},
			roleName: "developer",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := UserHasRole(tc.user, tc.roleName)
			if result != tc.expected {
				t.Errorf("Expected UserHasRole to return %v but got %v", tc.expected, result)
			}
		})
	}
}
