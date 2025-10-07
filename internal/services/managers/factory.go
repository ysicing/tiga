package managers

import (
	"fmt"
)

// ManagerFactory creates service managers based on service type
type ManagerFactory struct {
	managers map[string]func() ServiceManager
}

// NewManagerFactory creates a new manager factory
func NewManagerFactory() *ManagerFactory {
	factory := &ManagerFactory{
		managers: make(map[string]func() ServiceManager),
	}

	// Register all service managers
	factory.Register("minio", func() ServiceManager { return NewMinIOManager() })
	factory.Register("mysql", func() ServiceManager { return NewMySQLManager() })
	factory.Register("postgres", func() ServiceManager { return NewPostgreSQLManager() })
	factory.Register("redis", func() ServiceManager { return NewRedisManager() })
	factory.Register("docker", func() ServiceManager { return NewDockerManager() })
	factory.Register("k8s", func() ServiceManager { return NewK8sManager() })
	factory.Register("caddy", func() ServiceManager { return NewCaddyManager() })

	return factory
}

// Register registers a service manager creator
func (f *ManagerFactory) Register(serviceType string, creator func() ServiceManager) {
	f.managers[serviceType] = creator
}

// Create creates a service manager for the given type
func (f *ManagerFactory) Create(serviceType string) (ServiceManager, error) {
	creator, ok := f.managers[serviceType]
	if !ok {
		return nil, fmt.Errorf("unsupported service type: %s", serviceType)
	}

	return creator(), nil
}

// GetSupportedTypes returns all supported service types
func (f *ManagerFactory) GetSupportedTypes() []string {
	types := make([]string, 0, len(f.managers))
	for t := range f.managers {
		types = append(types, t)
	}
	return types
}
