package app

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/ysicing/tiga/internal/api"
	"github.com/ysicing/tiga/internal/api/middleware"
	"github.com/ysicing/tiga/internal/config"
	"github.com/ysicing/tiga/internal/db"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services"
	"github.com/ysicing/tiga/internal/services/alert"
	"github.com/ysicing/tiga/internal/services/auth"
	"github.com/ysicing/tiga/internal/services/host"
	"github.com/ysicing/tiga/internal/services/managers"
	"github.com/ysicing/tiga/internal/services/monitor"
	"github.com/ysicing/tiga/internal/services/notification"
	"github.com/ysicing/tiga/internal/services/scheduler"
	"github.com/ysicing/tiga/pkg/common"
	"github.com/ysicing/tiga/proto"

	installhandlers "github.com/ysicing/tiga/internal/install/handlers"
)

// Application represents the main application
type Application struct {
	config         *config.Config
	configPath     string
	db             *db.Database
	router         *middleware.RouterConfig
	scheduler      *scheduler.Scheduler
	coordinator    *managers.ManagerCoordinator
	httpServer     *http.Server
	grpcServer     *grpc.Server
	installMode    bool
	installChannel chan struct{}
	staticFS       embed.FS

	// Host monitoring services (shared between gRPC and HTTP)
	hostRepo        repository.HostRepository
	stateCollector  *host.StateCollector
	agentManager    *host.AgentManager
	terminalManager *host.TerminalManager
	hostService     *host.HostService

	// Service monitoring
	probeScheduler *monitor.ServiceProbeScheduler
}

// NewApplication creates a new application instance
func NewApplication(cfg *config.Config, configPath string, installMode bool, staticFS embed.FS) (*Application, error) {
	app := &Application{
		config:      cfg,
		configPath:  configPath,
		installMode: installMode,
		scheduler:   scheduler.NewScheduler(),
		staticFS:    staticFS,
	}

	// Skip database initialization in installation mode
	if !installMode {
		// Initialize database
		database, err := db.NewDatabase(&cfg.Database)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize database: %w", err)
		}

		// Auto migrate
		if err := database.AutoMigrate(); err != nil {
			return nil, fmt.Errorf("failed to migrate database: %w", err)
		}

		// Seed default data (groups, etc.)
		if err := database.SeedDefaultData(); err != nil {
			logrus.Warnf("Failed to seed default data: %v", err)
		}

		app.db = database
	}

	return app, nil
}

// Initialize initializes the application components
func (a *Application) Initialize(ctx context.Context) error {
	// In installation mode, skip full initialization
	if a.installMode {
		logrus.Info("Running in installation mode - skipping full initialization")
		return a.initializeInstallMode(ctx)
	}

	logrus.Info("Initializing application components...")

	// Initialize global DB for backward compatibility with some legacy packages
	models.DB = a.db.DB
	models.InitRepositories(a.db.DB)

	// Set encryption key from config for legacy code that uses common.GetEncryptKey()
	if a.config.Security.EncryptionKey != "" {
		// 验证加密密钥与数据库中存储的是否一致
		if err := a.validateEncryptionKey(); err != nil {
			return fmt.Errorf("encryption key validation failed: %w", err)
		}
		common.SetEncryptKey(a.config.Security.EncryptionKey)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(a.db.DB)
	instanceRepo := repository.NewInstanceRepository(a.db.DB)
	metricsRepo := repository.NewMetricsRepository(a.db.DB)
	alertRepo := repository.NewAlertRepository(a.db.DB)
	auditRepo := repository.NewAuditLogRepository(a.db.DB)

	// Initialize K8s repositories
	clusterRepo := repository.NewClusterRepository(a.db.DB)
	resourceHistoryRepo := repository.NewResourceHistoryRepository(a.db.DB)

	// Initialize K8s service
	k8sService := services.NewK8sService(clusterRepo, resourceHistoryRepo)

	// Try to import clusters from default kubeconfig
	if kubeconfigPath := os.Getenv("KUBECONFIG"); kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("HOME") + "/.kube/config"
		if _, err := os.Stat(kubeconfigPath); err == nil {
			if err := k8sService.ImportClustersFromKubeconfig(ctx, kubeconfigPath); err != nil {
				logrus.Warnf("Failed to import clusters from kubeconfig: %v", err)
			}
		}
	}

	// Initialize notification service
	notificationSvc := notification.NewNotificationService()

	// Register notifiers based on configuration
	// TODO: Load notifier configurations from database or config file

	// Initialize service managers
	managerFactory := managers.NewManagerFactory()
	a.coordinator = managers.NewManagerCoordinator(
		managerFactory,
		instanceRepo,
		metricsRepo,
		auditRepo,
	)

	// Initialize alert processor
	alertProcessor := alert.NewAlertProcessor(
		alertRepo,
		metricsRepo,
		notificationSvc,
		a.coordinator,
	)

	// Initialize service monitoring
	serviceRepo := repository.NewServiceRepository(a.db.DB)

	// Initialize AlertEngine for service monitoring alerts
	alertEngine := alert.NewAlertEngine(
		repository.NewMonitorAlertRepository(a.db.DB),
		a.hostRepo,
		serviceRepo,
	)

	// Setup scheduled tasks
	alertTask := scheduler.NewAlertTask(alertProcessor)
	a.scheduler.AddTask("alert_processing", alertTask, 30*time.Second)

	// Start monitoring
	a.coordinator.StartMonitoring(ctx)

	logrus.Info("Application components initialized successfully")

	// Get JWT secret from config (use a default if not set)
	jwtSecret := a.config.JWT.Secret
	if jwtSecret == "" {
		jwtSecret = "default-secret-change-in-production"
	}

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(
		jwtSecret,
		24*time.Hour,   // Access token duration
		7*24*time.Hour, // Refresh token duration
	)

	// Initialize JWT auth middleware
	middleware.InitJWTAuthMiddleware(jwtManager, a.db.DB)

	// Simplified: removed RBAC middleware initialization

	// Initialize host monitoring services (shared between gRPC and HTTP)
	serverURL := fmt.Sprintf("http://localhost:%d", a.config.Server.Port)
	a.hostRepo = repository.NewHostRepository(a.db.DB)
	a.stateCollector = host.NewStateCollector(a.hostRepo)
	a.agentManager = host.NewAgentManager(a.hostRepo, a.stateCollector, a.db.DB)
	a.stateCollector.SetAgentManager(a.agentManager) // Complete the circular reference
	a.terminalManager = host.NewTerminalManager()
	a.hostService = host.NewHostService(a.hostRepo, a.agentManager, a.stateCollector, serverURL)

	logrus.Info("Host monitoring services initialized")

	// Initialize service monitoring (ServiceProbeScheduler + ServiceSentinel)
	// Use serviceRepo and alertEngine initialized earlier
	a.probeScheduler = monitor.NewServiceProbeScheduler(serviceRepo, alertEngine)
	a.probeScheduler.SetAgentManager(a.agentManager) // Wire up agent task distribution

	logrus.Info("Service monitoring initialized")

	// Setup HTTP router
	routerConfig := &middleware.RouterConfig{
		DebugMode:       a.config.Server.Debug, // Set debug mode based on config
		EnableSwagger: true, // Enable Swagger UI
	}

	router := middleware.NewRouter(routerConfig)

	// Register all API handlers - pass host services and probe scheduler to avoid duplicate instances
	api.SetupRoutes(router, a.db.DB, a.configPath, jwtManager, jwtSecret, a.hostService, a.stateCollector, a.terminalManager, a.probeScheduler)

	// Serve static files from embedded filesystem
	a.setupStaticFiles(router)

	a.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", a.config.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(a.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(a.config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Initialize gRPC server for Agent communication - use shared services
	grpcService := host.NewGRPCServer(a.agentManager, a.terminalManager, a.probeScheduler)

	a.grpcServer = grpc.NewServer()
	proto.RegisterHostMonitorServer(a.grpcServer, grpcService)

	logrus.Info("gRPC server initialized for Agent monitoring")

	// Suppress unused variable warnings
	_ = userRepo

	return nil
}

// initializeInstallMode initializes minimal components for installation mode
func (a *Application) initializeInstallMode(_ context.Context) error {
	logrus.Info("Initializing installation mode...")

	// Create channel for installation completion signal first
	a.installChannel = make(chan struct{})

	// Setup HTTP router in installation mode
	routerConfig := &middleware.RouterConfig{
		DebugMode:       a.config.Server.Debug,
		EnableSwagger: false, // Disable Swagger in installation mode
	}

	router := middleware.NewRouter(routerConfig)

	// Register install handler with completion channel
	installHandler := installhandlers.NewInstallHandler(a.configPath, a.installChannel)
	apiGroup := router.Group("/api")
	installHandler.RegisterRoutes(apiGroup)

	// Serve static files from embedded filesystem
	a.setupStaticFiles(router)

	a.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", a.config.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(a.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(a.config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logrus.Info("Installation mode initialized successfully")
	return nil
}

// Run runs the application
func (a *Application) Run(ctx context.Context) error {
	// Start gRPC server for Agents (if not in install mode)
	if !a.installMode && a.grpcServer != nil {
		grpcPort := a.config.Server.GRPCPort
		if grpcPort == 0 {
			grpcPort = 12307
		}
		go func() {
			listener, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
			if err != nil {
				logrus.Fatalf("Failed to listen on gRPC port %d: %v", grpcPort, err)
			}
			logrus.Infof("Starting gRPC server on port %d", grpcPort)
			if err := a.grpcServer.Serve(listener); err != nil {
				logrus.Fatalf("gRPC server failed: %v", err)
			}
		}()
	}

	// Start HTTP server
	go func() {
		logrus.Infof("Starting HTTP server on %s", a.httpServer.Addr)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// In installation mode, wait for completion or interrupt
	if a.installMode {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		var installCompleted bool
		select {
		case <-a.installChannel:
			logrus.Info("Installation completed successfully")
			installCompleted = true
		case <-quit:
			logrus.Info("Installation interrupted by user")
		}

		// Shutdown HTTP server
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
			logrus.Errorf("Server forced to shutdown: %v", err)
		}

		// Auto-restart if installation completed successfully
		if installCompleted {
			logrus.Info("Restarting application in normal mode...")
			time.Sleep(1 * time.Second) // Brief pause for cleanup
			if err := a.restartProcess(); err != nil {
				logrus.Errorf("Failed to restart process: %v", err)
				logrus.Info("Please manually restart the application to enter normal mode.")
				return err
			}
		}

		logrus.Info("Installation server exited")
		return nil
	}

	// Normal mode: start services
	go a.scheduler.Start(ctx)

	// Start service probe scheduler (includes ServiceSentinel)
	if a.probeScheduler != nil {
		go a.probeScheduler.Start()
		logrus.Info("Service probe scheduler started")
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down server...")

	// Shutdown gRPC server
	if a.grpcServer != nil {
		logrus.Info("Stopping gRPC server...")
		a.grpcServer.GracefulStop()
	}

	// Shutdown HTTP server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		logrus.Errorf("Server forced to shutdown: %v", err)
	}

	// Stop scheduler
	a.scheduler.Stop()

	// Stop service probe scheduler (includes ServiceSentinel)
	if a.probeScheduler != nil {
		a.probeScheduler.Stop()
		logrus.Info("Service probe scheduler stopped")
	}

	// Stop monitoring
	a.coordinator.StopMonitoring()

	// Disconnect all managers
	a.coordinator.DisconnectAll(ctx)

	// Close database
	if err := a.db.Close(); err != nil {
		logrus.Errorf("Failed to close database: %v", err)
	}

	logrus.Info("Server exited")
	return nil
}

// validateEncryptionKey 验证配置文件中的加密密钥与数据库中存储的是否一致
func (a *Application) validateEncryptionKey() error {
	// 从数据库读取加密密钥
	var systemConfig struct {
		EncryptionKey string `gorm:"column:encryption_key"`
	}

	err := a.db.DB.Table("system_config").Select("encryption_key").First(&systemConfig).Error
	if err != nil {
		// 如果找不到记录，可能是首次启动或旧版本数据，记录警告但允许继续
		logrus.Warnf("Could not load encryption key from database: %v. This might be the first startup or legacy data.", err)
		return nil
	}

	// 比对配置文件和数据库中的加密密钥
	if systemConfig.EncryptionKey != a.config.Security.EncryptionKey {
		return fmt.Errorf("encryption key mismatch: config file encryption key does not match database. " +
			"Modifying encryption key after installation will break encrypted data. " +
			"If you need to change the encryption key, please use the key rotation procedure.")
	}

	logrus.Debug("Encryption key validation passed")
	return nil
}

// setupStaticFiles configures static file serving from embedded filesystem
func (a *Application) setupStaticFiles(router *gin.Engine) {
	// Serve assets directory
	assetsFS, err := fs.Sub(a.staticFS, "assets")
	if err != nil {
		logrus.Warnf("Failed to create assets sub-filesystem: %v", err)
	} else {
		router.StaticFS("/assets", http.FS(assetsFS))
	}

	// Serve index.html for all non-API routes (SPA catch-all)
	router.NoRoute(func(c *gin.Context) {
		data, err := a.staticFS.ReadFile("index.html")
		if err != nil {
			c.String(http.StatusNotFound, "404 page not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
}

// restartProcess restarts the current process
func (a *Application) restartProcess() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Get current process arguments
	args := os.Args

	// Get current environment
	env := os.Environ()

	// Execute the new process, replacing the current one
	// This works on Unix-like systems (Linux, macOS)
	return syscall.Exec(executable, args, env)
}
