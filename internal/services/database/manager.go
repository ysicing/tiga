package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	dbrepo "github.com/ysicing/tiga/internal/repository/database"
	"github.com/ysicing/tiga/pkg/crypto"
	"github.com/ysicing/tiga/pkg/dbdriver"
)

var (
	errUnsupportedDriver = errors.New("unsupported database driver type")
)

// CreateInstanceInput contains information required to register a database instance.
type CreateInstanceInput struct {
	Name        string
	Type        string
	Host        string
	Port        int
	Username    string
	Password    string
	SSLMode     string
	Description string
}

type cachedDriver struct {
	driver    dbdriver.DatabaseDriver
	updatedAt time.Time
}

// DatabaseManager provides lifecycle management and connection handling for database instances.
type DatabaseManager struct {
	instanceRepo *dbrepo.InstanceRepository

	cacheMu sync.RWMutex
	cache   map[uuid.UUID]cachedDriver
}

// NewDatabaseManager constructs a new DatabaseManager.
func NewDatabaseManager(instanceRepo *dbrepo.InstanceRepository) *DatabaseManager {
	return &DatabaseManager{
		instanceRepo: instanceRepo,
		cache:        make(map[uuid.UUID]cachedDriver),
	}
}

// CreateInstance validates, tests, encrypts, and persists a database instance.
func (m *DatabaseManager) CreateInstance(ctx context.Context, input CreateInstanceInput) (*models.DatabaseInstance, error) {
	if err := m.validateCreateInput(input); err != nil {
		return nil, err
	}

	// Test connection using plaintext credentials before persisting.
	if err := m.performConnectionTest(ctx, input); err != nil {
		return nil, fmt.Errorf("connection test failed: %w", err)
	}

	passwordCipher, err := encryptSecret(input.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt password: %w", err)
	}

	instance := &models.DatabaseInstance{
		Name:        input.Name,
		Type:        normalizeDriverType(input.Type),
		Host:        input.Host,
		Port:        input.Port,
		Username:    input.Username,
		Password:    passwordCipher,
		SSLMode:     input.SSLMode,
		Description: input.Description,
		Status:      "online",
		LastCheckAt: timePtr(time.Now().UTC()),
	}

	if err := m.instanceRepo.Create(ctx, instance); err != nil {
		return nil, err
	}

	// Hide secret from caller
	instance.Password = ""
	return instance, nil
}

// TestConnection verifies connectivity for the stored instance.
func (m *DatabaseManager) TestConnection(ctx context.Context, id uuid.UUID) error {
	instance, err := m.instanceRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	driver, cfg, err := m.prepareDriver(ctx, instance)
	if err != nil {
		return err
	}

	if err := driver.Connect(ctx, cfg); err != nil {
		return err
	}
	defer driver.Disconnect(ctx)

	return driver.Ping(ctx)
}

// GetInstance returns a single instance without exposing encrypted password.
func (m *DatabaseManager) GetInstance(ctx context.Context, id uuid.UUID) (*models.DatabaseInstance, error) {
	instance, err := m.instanceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	instance.Password = ""
	return instance, nil
}

// ListInstances returns all registered instances.
func (m *DatabaseManager) ListInstances(ctx context.Context) ([]*models.DatabaseInstance, error) {
	instances, err := m.instanceRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, inst := range instances {
		inst.Password = ""
	}
	return instances, nil
}

// DeleteInstance removes an instance and clears cached connections.
func (m *DatabaseManager) DeleteInstance(ctx context.Context, id uuid.UUID) error {
	m.cacheMu.Lock()
	if cached, ok := m.cache[id]; ok {
		_ = cached.driver.Disconnect(ctx)
		delete(m.cache, id)
	}
	m.cacheMu.Unlock()

	return m.instanceRepo.Delete(ctx, id)
}

// GetConnectedDriver returns an active driver for the given instance.
func (m *DatabaseManager) GetConnectedDriver(ctx context.Context, id uuid.UUID) (dbdriver.DatabaseDriver, *models.DatabaseInstance, error) {
	instance, err := m.instanceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	driver, err := m.getOrCreateDriver(ctx, instance)
	if err != nil {
		return nil, nil, err
	}

	return driver, instance, nil
}

func (m *DatabaseManager) getOrCreateDriver(ctx context.Context, instance *models.DatabaseInstance) (dbdriver.DatabaseDriver, error) {
	m.cacheMu.RLock()
	if cached, ok := m.cache[instance.ID]; ok {
		m.cacheMu.RUnlock()
		// ensure connection is still alive
		if err := cached.driver.Ping(ctx); err == nil && !instance.UpdatedAt.After(cached.updatedAt) {
			return cached.driver, nil
		}
		// recycle connection
		m.cacheMu.Lock()
		_ = cached.driver.Disconnect(ctx)
		delete(m.cache, instance.ID)
		m.cacheMu.Unlock()
	} else {
		m.cacheMu.RUnlock()
	}

	driver, cfg, err := m.prepareDriver(ctx, instance)
	if err != nil {
		return nil, err
	}

	if err := driver.Connect(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to connect to instance: %w", err)
	}

	m.cacheMu.Lock()
	m.cache[instance.ID] = cachedDriver{
		driver:    driver,
		updatedAt: instance.UpdatedAt,
	}
	m.cacheMu.Unlock()

	return driver, nil
}

func (m *DatabaseManager) prepareDriver(ctx context.Context, instance *models.DatabaseInstance) (dbdriver.DatabaseDriver, dbdriver.ConnectionConfig, error) {
	driver, err := createDriverForType(instance.Type)
	if err != nil {
		return nil, dbdriver.ConnectionConfig{}, err
	}

	password, err := decryptSecret(instance.Password)
	if err != nil {
		return nil, dbdriver.ConnectionConfig{}, fmt.Errorf("failed to decrypt stored password: %w", err)
	}

	cfg := dbdriver.ConnectionConfig{
		Host:         instance.Host,
		Port:         instance.Port,
		Username:     instance.Username,
		Password:     password,
		Database:     defaultDatabaseName(instance.Type),
		SSLMode:      instance.SSLMode,
		MaxOpenConns: 50,
		MaxIdleConns: 10,
	}

	return driver, cfg, nil
}

func (m *DatabaseManager) performConnectionTest(ctx context.Context, input CreateInstanceInput) error {
	driver, err := createDriverForType(input.Type)
	if err != nil {
		return err
	}

	cfg := dbdriver.ConnectionConfig{
		Host:         input.Host,
		Port:         input.Port,
		Username:     input.Username,
		Password:     input.Password,
		Database:     defaultDatabaseName(input.Type),
		SSLMode:      input.SSLMode,
		MaxOpenConns: 2,
		MaxIdleConns: 1,
	}

	if err := driver.Connect(ctx, cfg); err != nil {
		return err
	}
	defer driver.Disconnect(ctx)

	return driver.Ping(ctx)
}

func (m *DatabaseManager) validateCreateInput(input CreateInstanceInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("instance name is required")
	}
	if strings.TrimSpace(input.Type) == "" {
		return errors.New("instance type is required")
	}
	if strings.TrimSpace(input.Host) == "" {
		return errors.New("instance host is required")
	}
	if input.Port <= 0 {
		return errors.New("instance port must be positive")
	}
	if strings.TrimSpace(input.Username) == "" {
		return errors.New("instance username is required")
	}
	if input.Password == "" {
		return errors.New("instance password is required")
	}
	return nil
}

func createDriverForType(driverType string) (dbdriver.DatabaseDriver, error) {
	switch normalizeDriverType(driverType) {
	case "mysql":
		return dbdriver.NewMySQLDriver(), nil
	case "postgresql":
		return dbdriver.NewPostgresDriver(), nil
	case "redis":
		return dbdriver.NewRedisDriver(), nil
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedDriver, driverType)
	}
}

func normalizeDriverType(driverType string) string {
	switch strings.ToLower(strings.TrimSpace(driverType)) {
	case "postgres", "postgresql":
		return "postgresql"
	case "mysql":
		return "mysql"
	case "redis":
		return "redis"
	default:
		return driverType
	}
}

func encryptSecret(plaintext string) (string, error) {
	service := crypto.GetDefaultService()
	if service == nil {
		return "", errors.New("encryption service not initialised")
	}
	return service.Encrypt(plaintext)
}

func decryptSecret(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	service := crypto.GetDefaultService()
	if service == nil {
		return "", errors.New("encryption service not initialised")
	}
	return service.Decrypt(ciphertext)
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func defaultDatabaseName(driverType string) string {
	switch normalizeDriverType(driverType) {
	case "postgresql":
		return "postgres"
	default:
		return ""
	}
}
