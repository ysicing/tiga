package registry

import (
	"fmt"
	"sync"

	"github.com/ysicing/tiga/internal/services/base"
)

// Registry is the global service manager registry
var (
	globalRegistry *Registry
	once           sync.Once
)

// Registry manages all registered service managers
type Registry struct {
	managers map[string]base.ServiceManager
	mu       sync.RWMutex
}

// GetRegistry returns the global registry instance (singleton)
func GetRegistry() *Registry {
	once.Do(func() {
		globalRegistry = &Registry{
			managers: make(map[string]base.ServiceManager),
		}
	})
	return globalRegistry
}

// Register registers a service manager for a specific type
func (r *Registry) Register(serviceType string, manager base.ServiceManager) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if serviceType == "" {
		return fmt.Errorf("service type cannot be empty")
	}

	if manager == nil {
		return fmt.Errorf("service manager cannot be nil")
	}

	if _, exists := r.managers[serviceType]; exists {
		return fmt.Errorf("service manager for type %s is already registered", serviceType)
	}

	r.managers[serviceType] = manager
	return nil
}

// Unregister removes a service manager from the registry
func (r *Registry) Unregister(serviceType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.managers[serviceType]; !exists {
		return fmt.Errorf("service manager for type %s is not registered", serviceType)
	}

	delete(r.managers, serviceType)
	return nil
}

// Get retrieves a service manager by type
func (r *Registry) Get(serviceType string) (base.ServiceManager, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	manager, exists := r.managers[serviceType]
	if !exists {
		return nil, fmt.Errorf("service manager for type %s is not registered", serviceType)
	}

	return manager, nil
}

// GetAll returns all registered service managers
func (r *Registry) GetAll() map[string]base.ServiceManager {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modifications
	result := make(map[string]base.ServiceManager, len(r.managers))
	for k, v := range r.managers {
		result[k] = v
	}

	return result
}

// List returns all registered service types
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.managers))
	for serviceType := range r.managers {
		types = append(types, serviceType)
	}

	return types
}

// IsRegistered checks if a service type is registered
func (r *Registry) IsRegistered(serviceType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.managers[serviceType]
	return exists
}

// Count returns the number of registered service managers
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.managers)
}

// Clear removes all registered service managers (useful for testing)
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.managers = make(map[string]base.ServiceManager)
}

// Helper functions for global registry

// Register registers a service manager globally
func Register(serviceType string, manager base.ServiceManager) error {
	return GetRegistry().Register(serviceType, manager)
}

// Get retrieves a service manager globally
func Get(serviceType string) (base.ServiceManager, error) {
	return GetRegistry().Get(serviceType)
}

// List returns all registered service types globally
func List() []string {
	return GetRegistry().List()
}

// IsRegistered checks if a service type is registered globally
func IsRegistered(serviceType string) bool {
	return GetRegistry().IsRegistered(serviceType)
}

// ServiceType constants
const (
	ServiceTypeMinIO      = "minio"
	ServiceTypeMySQL      = "mysql"
	ServiceTypePostgreSQL = "postgresql"
	ServiceTypeRedis      = "redis"
	ServiceTypeDocker     = "docker"
	ServiceTypeKubernetes = "k8s"
	ServiceTypeCaddy      = "caddy"
)

// ValidServiceTypes returns all valid service types
func ValidServiceTypes() []string {
	return []string{
		ServiceTypeMinIO,
		ServiceTypeMySQL,
		ServiceTypePostgreSQL,
		ServiceTypeRedis,
		ServiceTypeDocker,
		ServiceTypeKubernetes,
		ServiceTypeCaddy,
	}
}

// IsValidServiceType checks if a service type is valid
func IsValidServiceType(serviceType string) bool {
	for _, valid := range ValidServiceTypes() {
		if serviceType == valid {
			return true
		}
	}
	return false
}
