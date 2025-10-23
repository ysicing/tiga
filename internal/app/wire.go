//go:build wireinject
// +build wireinject

package app

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/google/wire"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/config"
	"github.com/ysicing/tiga/internal/db"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services"
	"github.com/ysicing/tiga/internal/services/alert"
	"github.com/ysicing/tiga/internal/services/auth"
	dockerservices "github.com/ysicing/tiga/internal/services/docker"
	"github.com/ysicing/tiga/internal/services/host"
	"github.com/ysicing/tiga/internal/services/k8s"
	"github.com/ysicing/tiga/internal/services/managers"
	"github.com/ysicing/tiga/internal/services/monitor"
	"github.com/ysicing/tiga/internal/services/notification"
	"github.com/ysicing/tiga/internal/services/prometheus"
	"github.com/ysicing/tiga/internal/services/scheduler"
	"github.com/ysicing/tiga/pkg/kube"

	schedulerrepo "github.com/ysicing/tiga/internal/repository/scheduler"
)

// DatabaseSet provides database-related dependencies
var DatabaseSet = wire.NewSet(
	provideDatabaseConfig,
	db.NewDatabase,
	provideGormDB,
)

// RepositorySet provides all repository interfaces
var RepositorySet = wire.NewSet(
	repository.NewUserRepository,
	wire.Bind(new(repository.UserRepositoryInterface), new(*repository.UserRepository)),

	repository.NewInstanceRepository,
	wire.Bind(new(repository.InstanceRepositoryInterface), new(*repository.InstanceRepository)),

	repository.NewMetricsRepository,
	wire.Bind(new(repository.MetricsRepositoryInterface), new(*repository.MetricsRepository)),

	repository.NewAlertRepository,
	wire.Bind(new(repository.AlertRepositoryInterface), new(*repository.AlertRepository)),

	repository.NewAuditLogRepository,
	wire.Bind(new(repository.AuditLogRepositoryInterface), new(*repository.AuditLogRepository)),

	repository.NewClusterRepository,
	wire.Bind(new(repository.ClusterRepositoryInterface), new(*repository.ClusterRepository)),

	repository.NewResourceHistoryRepository,
	wire.Bind(new(repository.ResourceHistoryRepositoryInterface), new(*repository.ResourceHistoryRepository)),

	// Scheduler repositories
	schedulerrepo.NewExecutionRepository,
	schedulerrepo.NewTaskRepository,

	// T036-T037: 统一审计事件仓储（MinIO 和 Database 使用）
	repository.NewAuditEventRepository,

	// These return interfaces directly, no need for wire.Bind
	repository.NewHostRepository,
	repository.NewServiceRepository,
	repository.NewMonitorAlertRepository,
)

// ServiceSet provides core services
var ServiceSet = wire.NewSet(
	services.NewK8sService,
	notification.NewNotificationService,
	managers.NewManagerFactory,
	managers.NewManagerCoordinator,
	provideAlertProcessor,
	provideAlertEngine,
)

// HostMonitoringComponents aggregates host monitoring services to handle circular dependencies
type HostMonitoringComponents struct {
	StateCollector *host.StateCollector
	AgentManager   *host.AgentManager
}

// HostServiceSet provides host monitoring services
var HostServiceSet = wire.NewSet(
	provideHostMonitoringComponents, // Creates StateCollector and AgentManager with circular dependency
	host.NewTerminalManager,
	provideHostService,
	provideStateCollector, // Extract StateCollector from components
	provideAgentManager,   // Extract AgentManager from components
)

// DockerServiceSet provides Docker management services
// T032: Docker services needed for AgentManager integration
var DockerServiceSet = wire.NewSet(
	dockerservices.NewAgentForwarder,
	dockerservices.NewDockerInstanceService,
)

// MonitoringServiceSet provides monitoring services
var MonitoringServiceSet = wire.NewSet(
	provideServiceProbeScheduler,
)

// K8sServiceSet provides K8s cluster health and Prometheus discovery services
var K8sServiceSet = wire.NewSet(
	provideClientCache,
	prometheus.NewAutoDiscoveryService,
	k8s.NewClusterHealthService,
	// Phase 3 services
	k8s.NewCacheService,
	k8s.NewRelationsService,
	provideSearchService,
)

// AuthSet provides authentication services
var AuthSet = wire.NewSet(
	provideJWTManager,
)

// SchedulerSet provides scheduler
// T027: Updated to include ExecutionRepository dependency
var SchedulerSet = wire.NewSet(
	scheduler.NewScheduler,
)

// Provider functions to handle complex initialization

func provideDatabaseConfig(cfg *config.Config) *config.DatabaseConfig {
	return &cfg.Database
}

func provideGormDB(database *db.Database) *gorm.DB {
	return database.DB
}

func provideAlertProcessor(
	alertRepo *repository.AlertRepository,
	metricsRepo *repository.MetricsRepository,
	notificationSvc *notification.NotificationService,
	coordinator *managers.ManagerCoordinator,
) *alert.AlertProcessor {
	return alert.NewAlertProcessor(alertRepo, metricsRepo, notificationSvc, coordinator)
}

func provideAlertEngine(
	monitorAlertRepo repository.MonitorAlertRepository,
	hostRepo repository.HostRepository,
	serviceRepo repository.ServiceRepository,
) *alert.AlertEngine {
	return alert.NewAlertEngine(monitorAlertRepo, hostRepo, serviceRepo)
}

// Handle StateCollector and AgentManager circular dependency
// T038: Added AuditEventRepository for host audit logging
// T032: Added Docker services for instance lifecycle management
func provideHostMonitoringComponents(
	hostRepo repository.HostRepository,
	db *gorm.DB,
	auditEventRepo repository.AuditEventRepository,
	dockerInstanceService *dockerservices.DockerInstanceService,
	agentForwarder *dockerservices.AgentForwarder,
) *HostMonitoringComponents {
	// Create StateCollector first
	stateCollector := host.NewStateCollector(hostRepo)

	// T038: Create AuditLogger for host subsystem
	auditLogger := host.NewAuditLogger(auditEventRepo, nil)

	// T032: Create AgentManager with Docker integration
	agentManager := host.NewAgentManager(hostRepo, stateCollector, db, auditLogger, dockerInstanceService, agentForwarder)

	// Wire up the circular reference
	stateCollector.SetAgentManager(agentManager)

	return &HostMonitoringComponents{
		StateCollector: stateCollector,
		AgentManager:   agentManager,
	}
}

// Extract StateCollector from HostMonitoringComponents
func provideStateCollector(components *HostMonitoringComponents) *host.StateCollector {
	return components.StateCollector
}

// Extract AgentManager from HostMonitoringComponents
func provideAgentManager(components *HostMonitoringComponents) *host.AgentManager {
	return components.AgentManager
}

// T038: Provide AuditLogger for host subsystem (used by handlers)
func provideHostAuditLogger(auditEventRepo repository.AuditEventRepository) *host.AuditLogger {
	return host.NewAuditLogger(auditEventRepo, nil)
}

func provideHostService(
	hostRepo repository.HostRepository,
	agentManager *host.AgentManager,
	stateCollector *host.StateCollector,
	cfg *config.Config,
) *host.HostService {
	serverURL := fmt.Sprintf("http://localhost:%d", cfg.Server.Port)
	grpcPort := cfg.Server.GRPCPort
	if grpcPort == 0 {
		grpcPort = 12307
	}
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)

	return host.NewHostService(hostRepo, agentManager, stateCollector, serverURL, grpcAddr)
}

func provideServiceProbeScheduler(
	serviceRepo repository.ServiceRepository,
	alertEngine *alert.AlertEngine,
) *monitor.ServiceProbeScheduler {
	return monitor.NewServiceProbeScheduler(serviceRepo, alertEngine)
}

func provideClientCache() *kube.ClientCache {
	return kube.NewClientCache()
}

func provideSearchService(cacheService *k8s.CacheService) *k8s.SearchService {
	return k8s.NewSearchService(cacheService)
}

func provideJWTManager(cfg *config.Config) (*auth.JWTManager, error) {
	jwtSecret := cfg.JWT.Secret
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT secret is not configured")
	}
	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf("JWT secret must be at least 32 characters")
	}

	return auth.NewJWTManager(
		jwtSecret,
		24*time.Hour,   // Access token duration
		7*24*time.Hour, // Refresh token duration
	), nil
}

// InitializeApplication is the main wire injector function
func InitializeApplication(
	ctx context.Context,
	cfg *config.Config,
	configPath string,
	installMode bool,
	staticFS embed.FS,
) (*Application, error) {
	wire.Build(
		// Core sets
		DatabaseSet,
		RepositorySet,
		ServiceSet,
		HostServiceSet,
		DockerServiceSet, // T032: Docker services for AgentManager
		MonitoringServiceSet,
		K8sServiceSet,
		AuthSet,
		SchedulerSet,

		// Application constructor
		newWireApplication,
	)

	return nil, nil // Wire generates the actual implementation
}

// newWireApplication is the internal constructor used by wire
func newWireApplication(
	cfg *config.Config,
	configPath string,
	installMode bool,
	staticFS embed.FS,
	database *db.Database,
	scheduler *scheduler.Scheduler,
	coordinator *managers.ManagerCoordinator,
	jwtManager *auth.JWTManager,
	hostRepo repository.HostRepository,
	stateCollector *host.StateCollector,
	agentManager *host.AgentManager,
	terminalManager *host.TerminalManager,
	hostService *host.HostService,
	probeScheduler *monitor.ServiceProbeScheduler,
	k8sService *services.K8sService,
	clusterHealthService *k8s.ClusterHealthService,
	prometheusDiscovery *prometheus.AutoDiscoveryService,
	// Phase 3 services
	cacheService *k8s.CacheService,
	relationsService *k8s.RelationsService,
	searchService *k8s.SearchService,
) (*Application, error) {
	app := &Application{
		config:               cfg,
		configPath:           configPath,
		db:                   database,
		scheduler:            scheduler,
		coordinator:          coordinator,
		jwtManager:           jwtManager,
		installMode:          installMode,
		staticFS:             staticFS,
		hostRepo:             hostRepo,
		stateCollector:       stateCollector,
		agentManager:         agentManager,
		terminalManager:      terminalManager,
		hostService:          hostService,
		probeScheduler:       probeScheduler,
		k8sService:           k8sService,
		clusterHealthService: clusterHealthService,
		prometheusDiscovery:  prometheusDiscovery,
		// Phase 3 services
		cacheService:     cacheService,
		relationsService: relationsService,
		searchService:    searchService,
	}

	return app, nil
}
