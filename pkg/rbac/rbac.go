package rbac

import (
	"fmt"
	"slices"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/pkg/common"
)

// CanAccess checks if user can access resource with verb in cluster/namespace
// Simplified: Admin users have full access, regular users have read-only access
func CanAccess(user models.User, resource, verb, cluster, namespace string) bool {
	// Admin users have full access
	if user.IsAdmin {
		logrus.Debugf("RBAC Check - User: %s (Admin), Resource: %s, Verb: %s, Cluster: %s, Namespace: %s, Access: true",
			user.Key(), resource, verb, cluster, namespace)
		return true
	}

	// Regular users can only perform read operations
	readVerbs := []string{"get", "list", "log"}
	isReadVerb := false
	for _, v := range readVerbs {
		if v == verb {
			isReadVerb = true
			break
		}
	}

	logrus.Debugf("RBAC Check - User: %s (Regular), Resource: %s, Verb: %s, Cluster: %s, Namespace: %s, Access: %t",
		user.Key(), resource, verb, cluster, namespace, isReadVerb)
	return isReadVerb
}

func CanAccessCluster(user models.User, name string) bool {
	// Simplified: All authenticated users can access all clusters
	// Access control is handled by verb-level checks in CanAccess
	return true
}

func CanAccessNamespace(user models.User, cluster, name string) bool {
	// Simplified: All authenticated users can access all namespaces
	// Access control is handled by verb-level checks in CanAccess
	return true
}

// GetUserRoles returns roles for a user (simplified - returns empty as we use is_admin now)
func GetUserRoles(user models.User) []common.Role {
	// Simplified: RBAC system removed, using is_admin flag instead
	// This function kept for backwards compatibility but returns empty roles
	return []common.Role{}
}

func findRole(name string) *common.Role {
	rwlock.RLock()
	defer rwlock.RUnlock()
	for _, r := range RBACConfig.Roles {
		if r.Name == name {
			return &r
		}
	}
	return nil
}

func match(list []string, val string) bool {
	for _, v := range list {
		if len(v) > 1 && strings.HasPrefix(v, "!") {
			if v[1:] == val {
				return false
			}
		}
		if v == "*" || v == val {
			return true
		}
	}
	return false
}

func contains(list []string, val string) bool {
	return slices.Contains(list, val)
}

func NoAccess(user, verb, resource, ns, cluster string) string {
	if ns == "" {
		return fmt.Sprintf("user %s does not have permission to %s %s on cluster %s",
			user, verb, resource, cluster)
	}
	if ns == "_all" {
		ns = "All"
	}
	return fmt.Sprintf("user %s does not have permission to %s %s in namespace %s on cluster %s",
		user, verb, resource, ns, cluster)
}

func UserHasRole(user models.User, roleName string) bool {
	// Simplified: Check if user is admin when roleName is "admin"
	// All other role checks return false as role system is removed
	if roleName == "admin" {
		return user.IsAdmin
	}
	return false
}
