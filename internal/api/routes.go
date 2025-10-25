package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/api/handlers/instances"
	"github.com/ysicing/tiga/internal/api/handlers/minio"
	"github.com/ysicing/tiga/internal/api/middleware"
	"github.com/ysicing/tiga/internal/config"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services"
	"github.com/ysicing/tiga/pkg/auth"
	"github.com/ysicing/tiga/pkg/cluster"
	"github.com/ysicing/tiga/pkg/handlers/resources"

	audithandlers "github.com/ysicing/tiga/internal/api/handlers/audit"
	clusterhandlers "github.com/ysicing/tiga/internal/api/handlers/cluster"
	databasehandlers "github.com/ysicing/tiga/internal/api/handlers/database"
	dockerhandlers "github.com/ysicing/tiga/internal/api/handlers/docker"
	schedulerhandlers "github.com/ysicing/tiga/internal/api/handlers/scheduler"
	installhandlers "github.com/ysicing/tiga/internal/install/handlers"
	dbrepo "github.com/ysicing/tiga/internal/repository/database"
	schedulerrepo "github.com/ysicing/tiga/internal/repository/scheduler"
	alertservices "github.com/ysicing/tiga/internal/services/alert"
	authservices "github.com/ysicing/tiga/internal/services/auth"
	dbservices "github.com/ysicing/tiga/internal/services/database"
	dockerservices "github.com/ysicing/tiga/internal/services/docker"
	hostservices "github.com/ysicing/tiga/internal/services/host"
	monitorservices "github.com/ysicing/tiga/internal/services/monitor"
	schedulerservices "github.com/ysicing/tiga/internal/services/scheduler"
	websshservices "github.com/ysicing/tiga/internal/services/webssh"
	pkghandlers "github.com/ysicing/tiga/pkg/handlers"
	pkgmiddleware "github.com/ysicing/tiga/pkg/middleware"
)

// SetupRoutes configures all application routes
func SetupRoutes(
	router *gin.Engine,
	db *gorm.DB,
	configPath string,
	jwtManager *authservices.JWTManager,
	jwtSecret string,
	dbManagementCfg config.DatabaseManagementConfig,
	hostService *hostservices.HostService,
	stateCollector *hostservices.StateCollector,
	terminalManager *hostservices.TerminalManager,
	dockerStreamManager *hostservices.DockerStreamManager,
	probeScheduler *monitorservices.ServiceProbeScheduler,
	agentManager *hostservices.AgentManager,
	cfg *config.Config,
) {
	// Initialize global JWT auth middleware
	middleware.InitJWTAuthMiddleware(jwtManager, db)

	// Global middleware to inject config into context for all routes
	router.Use(middleware.ConfigMiddleware(cfg))

	// Global middleware to inject DB into context for all routes
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	// ==================== Install Routes (Only if not installed) ====================
	// Check if system is installed
	configService := config.NewInstallConfigService(configPath)
	if !configService.IsInstalled() {
		// Register installation wizard routes (only when not installed)
		installHandler := installhandlers.NewInstallHandler(configPath, nil)
		apiGroup := router.Group("/api")
		installHandler.RegisterRoutes(apiGroup)

		// Skip all other routes in installation mode
		return
	}

	// ==================== Normal Mode Routes ====================

	// Initialize cluster manager
	clusterManager, err := cluster.NewClusterManager()
	if err != nil {
		panic("Failed to create ClusterManager: " + err.Error())
	}

	// Initialize repositories
	instanceRepo := repository.NewInstanceRepository(db)
	alertRepo := repository.NewAlertRepository(db)
	auditRepo := repository.NewAuditLogRepository(db)

	// System user management repository
	userRepo := repository.NewUserRepository(db)

	// K8s repositories (Phase 0-3)
	clusterRepo := repository.NewClusterRepository(db)
	resourceHistoryRepo := repository.NewResourceHistoryRepository(db)

	// Host monitoring repositories
	hostRepo := repository.NewHostRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	monitorAlertRepo := repository.NewMonitorAlertRepository(db)

	// Database management repositories
	dbInstanceRepo := dbrepo.NewInstanceRepository(db)
	dbDatabaseRepo := dbrepo.NewDatabaseRepository(db)
	dbUserRepo := dbrepo.NewUserRepository(db)
	dbPermissionRepo := dbrepo.NewPermissionRepository(db)
	dbQuerySessionRepo := dbrepo.NewQuerySessionRepository(db)

	// Docker management repositories
	dockerInstanceRepo := repository.NewDockerInstanceRepository(db)
	terminalRecordingRepo := repository.NewTerminalRecordingRepository(db)

	// Scheduler repositories (T013, T019)
	schedulerTaskRepo := schedulerrepo.NewTaskRepository(db)
	schedulerExecutionRepo := schedulerrepo.NewExecutionRepository(db)

	// Audit repositories (T012, T020)
	auditEventRepo := repository.NewAuditEventRepository(db)

	// Initialize session and login services using the shared jwtManager
	sessionService := authservices.NewSessionService(db)
	loginService := authservices.NewLoginService(db, jwtManager, sessionService)

	// System user management service
	rbacService := authservices.NewRBACService(db)

	// Initialize services
	instanceService := services.NewInstanceService(instanceRepo)

	// Database management services
	dbManager := dbservices.NewDatabaseManager(dbInstanceRepo)
	dbSecurityFilter := dbservices.NewSecurityFilter()
	dbDatabaseService := dbservices.NewDatabaseService(dbManager, dbDatabaseRepo)
	dbUserService := dbservices.NewUserService(dbManager, dbUserRepo)
	dbPermissionService := dbservices.NewPermissionService(dbManager, dbUserRepo, dbDatabaseRepo, dbPermissionRepo)
	dbAuditLogger := dbservices.NewAuditLogger(auditEventRepo) // T036-T037: 使用统一的 AuditEventRepository
	dbQueryExecutor := dbservices.NewQueryExecutorWithConfig(
		dbManager,
		dbQuerySessionRepo,
		dbSecurityFilter,
		&dbservices.QueryExecutorConfig{
			Timeout:        dbManagementCfg.QueryTimeout(),
			MaxResultBytes: dbManagementCfg.ResultSizeLimit(),
		},
	)

	// Docker management services
	dockerAgentForwarder := dockerservices.NewAgentForwarderV2(agentManager, db)  // Use task-queue based forwarder
	dockerInstanceService := dockerservices.NewDockerInstanceService(db)
	dockerContainerService := dockerservices.NewContainerService(db, dockerInstanceService, dockerAgentForwarder)
	dockerImageService := dockerservices.NewImageService(db, dockerInstanceService, dockerAgentForwarder)
	dockerAuditService := dockerservices.NewAuditLogService(auditRepo)
	// Unused services for future phases
	_ = dockerservices.NewDockerHealthService(dockerInstanceRepo, dockerAgentForwarder, db)
	_ = dockerservices.NewDockerCacheService()

	// Host monitoring services - use shared instances from app.go to avoid duplicate creation
	// stateCollector, hostService, terminalManager, and probeScheduler are passed as parameters
	probeService := monitorservices.NewServiceProbeService(serviceRepo, probeScheduler)
	sessionManager := websshservices.NewSessionManager(db, "./data/recordings")

	// agentManager is already passed as a parameter, no need to get it from hostService

	// Start background services
	_ = alertservices.NewAlertEngine(monitorAlertRepo, hostRepo, serviceRepo) // Runs in background (monitor alerts)
	expiryScheduler := hostservices.NewExpiryScheduler(hostRepo, monitorAlertRepo, db)
	expiryScheduler.Start()

	// Scheduler service (T013)
	schedulerService := schedulerservices.NewScheduler(schedulerTaskRepo, schedulerExecutionRepo)
	schedulerService.Start(context.Background()) // Start the scheduler background worker

	// T027: Register existing tasks with Scheduler
	// Note: alert_processing task uses AlertProcessor (for database alerts), not AlertEngine (for monitor alerts)
	// Commented out until AlertProcessor dependencies are available in routes.go
	// TODO: Uncomment when notificationSvc and coordinator are available
	// alertProcessor := alertservices.NewAlertProcessor(alertRepo, metricsRepo, notificationSvc, coordinator)
	// alertTask := schedulerservices.NewAlertTask(alertProcessor)
	// if err := schedulerService.AddCron(
	// 	"alert_processing",
	// 	"*/5 * * * *", // Every 5 minutes
	// 	alertTask,
	// ); err != nil {
	// 	logrus.Errorf("Failed to register alert_processing task: %v", err)
	// } else {
	// 	logrus.Info("alert_processing task registered successfully")
	// }

	// 2. Database audit cleanup task (daily at 2 AM)
	// TODO: Get retention days from config (default: 90 days)
	// T036-T037: 使用统一的 AuditEventRepository
	dbAuditCleanupTask := schedulerservices.NewDatabaseAuditCleanupTask(auditEventRepo, 90)
	if err := schedulerService.AddCron(
		"database_audit_cleanup",
		"0 2 * * *", // Daily at 2:00 AM
		dbAuditCleanupTask,
	); err != nil {
		logrus.Errorf("Failed to register database_audit_cleanup task: %v", err)
	} else {
		logrus.Info("database_audit_cleanup task registered successfully")
	}

	// 3. Terminal recording cleanup task (daily at 3 AM)
	// Deletes terminal recordings older than 90 days (both DB records and files)
	terminalRecordingCleanupTask := schedulerservices.NewTerminalRecordingCleanupTask(terminalRecordingRepo, 90)
	if err := schedulerService.AddCron(
		"terminal_recording_cleanup",
		"0 3 * * *", // Daily at 3:00 AM
		terminalRecordingCleanupTask,
	); err != nil {
		logrus.Errorf("Failed to register terminal_recording_cleanup task: %v", err)
	} else {
		logrus.Info("terminal_recording_cleanup task registered successfully")
	}

	// Initialize handlers
	instanceHandler := handlers.NewInstanceHandler(instanceRepo)
	healthHandler := instances.NewHealthHandler(instanceService)
	metricsHandler := instances.NewMetricsHandler(instanceService)
	alertHandler := handlers.NewAlertHandler(alertRepo)
	auditHandler := handlers.NewAuditLogHandler(auditRepo)

	// Scheduler handlers (T022)
	schedulerTaskHandler := schedulerhandlers.NewTaskHandler(schedulerTaskRepo, schedulerService)
	schedulerExecutionHandler := schedulerhandlers.NewExecutionHandler(schedulerExecutionRepo, schedulerTaskRepo)
	schedulerStatsCalculator := schedulerservices.NewStatsCalculator(schedulerExecutionRepo)
	schedulerStatsHandler := schedulerhandlers.NewStatsHandler(schedulerStatsCalculator)

	// Audit handlers (T023)
	auditEventHandler := audithandlers.NewEventHandler(auditEventRepo)
	auditConfigHandler := audithandlers.NewConfigHandler()

	// K8s handlers (Phase 0-4)
	k8sClusterHandler := clusterhandlers.NewClusterHandler(clusterRepo, resourceHistoryRepo, cfg)

	// Database management handlers
	dbInstanceHandler := databasehandlers.NewInstanceHandler(dbManager, dbAuditLogger)
	dbDatabaseHandler := databasehandlers.NewDatabaseHandler(dbDatabaseService, dbAuditLogger)
	dbUserHandler := databasehandlers.NewUserHandler(dbUserService, dbAuditLogger)
	dbPermissionHandler := databasehandlers.NewPermissionHandler(dbPermissionService, dbAuditLogger)
	dbQueryHandler := databasehandlers.NewQueryHandler(dbQueryExecutor, dbAuditLogger)
	// T036-T037: 审计 API 已统一到 /api/v1/audit，移除旧的 dbAuditHandler

	// Docker management handlers
	dockerInstanceHandler := dockerhandlers.NewInstanceHandler(dockerInstanceService, dockerAgentForwarder)
	dockerContainerHandler := dockerhandlers.NewContainerHandler(dockerContainerService, dockerAgentForwarder)
	dockerStatsHandler := dockerhandlers.NewContainerStatsHandler(dockerStreamManager, agentManager, db)
	dockerLogsHandler := dockerhandlers.NewContainerLogsHandler(dockerStreamManager, agentManager, db)
	dockerImageHandler := dockerhandlers.NewImageHandler(dockerImageService, dockerAgentForwarder)
	dockerAuditHandler := dockerhandlers.NewAuditLogHandler(dockerAuditService)
	dockerVolumeHandler := dockerhandlers.NewVolumeHandler(dockerAgentForwarder)
	dockerNetworkHandler := dockerhandlers.NewNetworkHandler(dockerAgentForwarder)
	dockerSystemHandler := dockerhandlers.NewSystemHandler(dockerAgentForwarder)

	// Terminal handlers (using terminalRecordingRepo created earlier)
	dockerTerminalHandler := dockerhandlers.NewTerminalHandler(db, dockerStreamManager, agentManager, dockerInstanceService, jwtManager, terminalRecordingRepo)
	dockerRecordingHandler := dockerhandlers.NewRecordingHandler(db, terminalRecordingRepo)

	// Host monitoring handlers
	hostHandler := handlers.NewHostHandler(hostService)
	hostGroupHandler := handlers.NewHostGroupHandler(db)
	serviceMonitorHandler := handlers.NewServiceMonitorHandler(probeService)
	// T038: hostActivityHandler 已移除，使用统一审计 API: /api/v1/audit/events?subsystem=host
	monitorAlertHandler := handlers.NewMonitorAlertRuleHandler(monitorAlertRepo)

	// T038: Create host audit logger for handlers
	hostAuditLogger := hostservices.NewAuditLogger(auditEventRepo, nil)
	websshHandler := handlers.NewWebSSHHandler(sessionManager, terminalManager, agentManager, db, hostAuditLogger)
	websocketHandler := handlers.NewWebSocketHandler(stateCollector)

	// MinIO handlers
	minioBucketHandler := minio.NewBucketHandler(*instanceRepo)
	minioObjectHandler := minio.NewObjectHandler(*instanceRepo)

	// Auth handler for /api/auth routes - using new system
	authHandler := handlers.NewAuthHandler(loginService, sessionService, nil)

	// System user management handler
	userHandler := handlers.NewUserHandler(userRepo, rbacService)

	// ==================== Auth Routes (No Auth Required /api/auth) ====================
	// Note: These are at /api/auth (not /api/v1/auth) for compatibility with frontend
	authGroup := router.Group("/api/auth")
	{
		// Get available OAuth providers (no auth required)
		authGroup.GET("/providers", authHandler.GetOAuthProviders)
		// Password login endpoint - using new system
		authGroup.POST("/login/password", authHandler.Login)
		// Refresh token endpoint (no auth required - uses refresh token)
		authGroup.POST("/refresh", authHandler.RefreshToken)
	}

	// ==================== Public Config API (No Auth Required) ====================
	// Get app configuration (app name, subtitle, etc.) - used by login page
	router.GET("/api/config", handlers.GetAppConfig(configPath))

	// System configuration handler
	systemHandler := handlers.NewSystemHandler(db)

	// Public system configuration (no auth required) - used by frontend
	router.GET("/api/system/config", systemHandler.GetPublicConfig)

	// Auth routes requiring authentication
	authProtected := router.Group("/api/auth")
	authProtected.Use(middleware.AuthRequired())
	{
		// Get current user info
		authProtected.GET("/user", authHandler.GetCurrentUser)
		// Logout
		authProtected.POST("/logout", authHandler.Logout)
	}

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		// ==================== Setup & Initialization (No Auth Required) ====================
		// Initialization check endpoint (no auth required)
		v1.GET("/init_check", pkghandlers.NewInitCheckHandler(configPath))

		// App configuration endpoint (no auth required) - used by login page
		v1.GET("/config", handlers.GetAppConfig(configPath))

		adminAPI := v1.Group("/admin")
		{
			// Initial setup endpoints (no auth required - only works when no users exist)
			adminAPI.POST("/users/create_super_user", pkghandlers.CreateSuperUser)
		}

		// ==================== Kubernetes Subsystem ====================
		// Cluster list endpoint (requires basic auth)
		v1.GET("/clusters", middleware.AuthRequired(), clusterManager.GetClusters)

		// Kubernetes cluster management (admin only)
		k8sAdminAPI := adminAPI.Group("/clusters")
		k8sAdminAPI.Use(middleware.AuthRequired(), middleware.RequireAdmin())
		{
			k8sAdminAPI.GET("/", clusterManager.GetClusterList)
			k8sAdminAPI.POST("/", clusterManager.CreateCluster)
			k8sAdminAPI.POST("/import", clusterManager.ImportClustersFromKubeconfig)
			k8sAdminAPI.PUT("/:id", clusterManager.UpdateCluster)
			k8sAdminAPI.DELETE("/:id", clusterManager.DeleteCluster)
		}

		// ==================== User & Auth Subsystem ====================
		// User management (admin only)
		userAdminAPI := adminAPI.Group("/users")
		userAdminAPI.Use(middleware.AuthRequired(), middleware.RequireAdmin())
		{
			userAdminAPI.GET("/", pkghandlers.ListUsers)
			userAdminAPI.POST("/", pkghandlers.CreatePasswordUser)
			userAdminAPI.PUT("/:id", pkghandlers.UpdateUser)
			userAdminAPI.DELETE("/:id", pkghandlers.DeleteUser)
			userAdminAPI.POST("/:id/reset_password", pkghandlers.ResetPassword)
			userAdminAPI.POST("/:id/enable", pkghandlers.SetUserEnabled)
		}

		// OAuth provider management (admin only)
		oauthAdminAPI := adminAPI.Group("/oauth-providers")
		oauthAdminAPI.Use(middleware.AuthRequired(), middleware.RequireAdmin())
		{
			authHandler := auth.NewAuthHandler(db, jwtSecret)
			oauthAdminAPI.GET("/", authHandler.ListOAuthProviders)
			oauthAdminAPI.POST("/", authHandler.CreateOAuthProvider)
			oauthAdminAPI.GET("/:id", authHandler.GetOAuthProvider)
			oauthAdminAPI.PUT("/:id", authHandler.UpdateOAuthProvider)
			oauthAdminAPI.DELETE("/:id", authHandler.DeleteOAuthProvider)
		}

		// System configuration management (admin only)
		systemAdminAPI := adminAPI.Group("/system")
		systemAdminAPI.Use(middleware.AuthRequired(), middleware.RequireAdmin())
		{
			systemAdminAPI.GET("/config", systemHandler.GetSystemConfig)
			systemAdminAPI.PUT("/config", systemHandler.UpdateSystemConfig)
		}

		// ==================== Protected Endpoints (Require Auth) ====================
		protected := v1.Group("")
		protected.Use(middleware.AuthRequired(), middleware.ReadonlyMode(cfg))
		{
			// ==================== New K8s Cluster Management API (Phase 0-4) ====================
			k8sGroup := protected.Group("/k8s")
			{
				// Unified CRD status check (OpenKruise, Tailscale, Traefik, SystemUpgrade)
				crdStatusHandler := pkghandlers.NewCRDStatusHandler()
				k8sGroup.Use(pkgmiddleware.ClusterMiddleware(clusterManager))
				k8sGroup.GET("/crd-status/:type", crdStatusHandler.GetCRDStatus)

				// Cluster management
				clustersGroup := k8sGroup.Group("/clusters")
				{
					clustersGroup.GET("", k8sClusterHandler.List)
					clustersGroup.POST("", k8sClusterHandler.Create)
					clustersGroup.GET("/:id", k8sClusterHandler.Get)
					clustersGroup.PUT("/:id", k8sClusterHandler.Update)
					clustersGroup.DELETE("/:id", k8sClusterHandler.Delete)
					clustersGroup.POST("/:id/test", k8sClusterHandler.TestConnection)
					clustersGroup.POST("/:id/set-default", k8sClusterHandler.SetDefault)

					// Prometheus discovery (Phase 1)
					clustersGroup.POST("/:id/prometheus/rediscover", k8sClusterHandler.RediscoverPrometheus)

					// CRD detection (Phase 2)
					clustersGroup.GET("/:id/crds", k8sClusterHandler.DetectCRDs)

					// CloneSet operations (Phase 2)
					clustersGroup.GET("/:id/clonesets", k8sClusterHandler.ListCloneSets)
					clustersGroup.PUT("/:id/clonesets/:name/scale", k8sClusterHandler.ScaleCloneSet)
					clustersGroup.POST("/:id/clonesets/:name/restart", k8sClusterHandler.RestartCloneSet)

					// DaemonSet operations (Phase 2)
					clustersGroup.GET("/:id/daemonsets", k8sClusterHandler.ListDaemonSets)
					clustersGroup.POST("/:id/daemonsets/:name/restart", k8sClusterHandler.RestartDaemonSet)

					// StatefulSet operations (Phase 2)
					clustersGroup.GET("/:id/statefulsets", k8sClusterHandler.ListStatefulSets)
					clustersGroup.PUT("/:id/statefulsets/:name/scale", k8sClusterHandler.ScaleStatefulSet)
					clustersGroup.POST("/:id/statefulsets/:name/restart", k8sClusterHandler.RestartStatefulSet)

					// Resource history (Phase 3)
					clustersGroup.GET("/:id/resource-history", k8sClusterHandler.ListResourceHistory)
					clustersGroup.GET("/:id/resource-history/:history_id", k8sClusterHandler.GetResourceHistory)
					clustersGroup.DELETE("/:id/resource-history/:history_id", k8sClusterHandler.DeleteResourceHistory)

					// Generic CRD CRUD (Phase 3)
					clustersGroup.GET("/:id/crd-resources", k8sClusterHandler.ListCRDResources)
					clustersGroup.GET("/:id/crd-resources/:name", k8sClusterHandler.GetCRDResource)
					clustersGroup.POST("/:id/crd-resources", k8sClusterHandler.CreateCRDResource)
					clustersGroup.PUT("/:id/crd-resources/:name", k8sClusterHandler.UpdateCRDResource)
					clustersGroup.DELETE("/:id/crd-resources/:name", k8sClusterHandler.DeleteCRDResource)
					clustersGroup.POST("/:id/crd-resources/apply", k8sClusterHandler.ApplyCRDResourceYAML)

					// WebSocket terminal (Phase 4)
					clustersGroup.GET("/:id/terminal", k8sClusterHandler.PodTerminal)
				}
			}

			// ==================== Kubernetes Resources & Operations ====================
			// K8s resources (requires cluster middleware)
			// All K8s routes are under /cluster/:clusterid prefix
			clusterGroup := protected.Group("/cluster/:clusterid")
			clusterGroup.Use(pkgmiddleware.ClusterMiddleware(clusterManager), pkgmiddleware.RBACMiddleware())
			{
				// K8s Overview
				clusterGroup.GET("/overview", pkghandlers.GetOverview)

				// Prometheus metrics
				promHandler := pkghandlers.NewPromHandler()
				clusterGroup.GET("/prometheus/resource-usage-history", promHandler.GetResourceUsageHistory)
				clusterGroup.GET("/prometheus/pods/:namespace/:podName/metrics", promHandler.GetPodMetrics)

				// WebSocket endpoints for logs and terminals
				logsHandler := pkghandlers.NewLogsHandler()
				clusterGroup.GET("/logs/:namespace/:podName/ws", logsHandler.HandleLogsWebSocket)

				terminalHandler := pkghandlers.NewTerminalHandler()
				clusterGroup.GET("/terminal/:namespace/:podName/ws", terminalHandler.HandleTerminalWebSocket)

				nodeTerminalHandler := pkghandlers.NewNodeTerminalHandler()
				clusterGroup.GET("/node-terminal/:nodeName/ws", nodeTerminalHandler.HandleNodeTerminalWebSocket)

				// Search and resource operations
				searchHandler := pkghandlers.NewSearchHandler()
				clusterGroup.GET("/search", searchHandler.GlobalSearch)

				resourceApplyHandler := pkghandlers.NewResourceApplyHandler()
				clusterGroup.POST("/resources/apply", resourceApplyHandler.ApplyResource)

				clusterGroup.GET("/image/tags", pkghandlers.GetImageTags)

				// K8s Resource CRUD routes (Pods, Deployments, Services, etc.)
				// This will register routes like:
				// /api/v1/cluster/:clusterid/pods
				// /api/v1/cluster/:clusterid/deployments
				// /api/v1/cluster/:clusterid/rolebindings
				// etc.
				resources.RegisterRoutes(clusterGroup)
			}

			// ==================== Database Instance Management Subsystem ====================
			registerDatabaseRoutes := func(group *gin.RouterGroup) {
				// Database instance CRUD
				group.GET("", instanceHandler.ListInstances)
				group.POST("", instanceHandler.CreateInstance)
				group.GET("/:instance_id", instanceHandler.GetInstance)
				group.PUT("/:instance_id", instanceHandler.UpdateInstance)
				group.DELETE("/:instance_id", instanceHandler.DeleteInstance)

				// Instance health and metrics
				group.GET("/:instance_id/health", healthHandler.GetInstanceHealth)
				group.GET("/:instance_id/metrics", metricsHandler.GetInstanceMetrics)
			}

			// Primary database routes
			registerDatabaseRoutes(protected.Group("/dbs"))
			// Legacy routes for backward compatibility
			registerDatabaseRoutes(protected.Group("/instances"))

			// ==================== New Database Management Subsystem ====================
			databaseGroup := protected.Group("/database")
			databaseGroup.Use(middleware.RequireAdmin())
			{
				instancesGroup := databaseGroup.Group("/instances")
				{
					instancesGroup.GET("", dbInstanceHandler.ListInstances)
					instancesGroup.POST("", dbInstanceHandler.CreateInstance)
					instancesGroup.GET("/:id", dbInstanceHandler.GetInstance)
					instancesGroup.DELETE("/:id", dbInstanceHandler.DeleteInstance)
					instancesGroup.POST("/:id/test", dbInstanceHandler.TestConnection)
				}

				databasesGroup := databaseGroup.Group("/instances/:id/databases")
				{
					databasesGroup.GET("", dbDatabaseHandler.ListDatabases)
					databasesGroup.POST("", dbDatabaseHandler.CreateDatabase)
				}
				databaseGroup.DELETE("/databases/:id", dbDatabaseHandler.DeleteDatabase)

				usersGroup := databaseGroup.Group("/instances/:id/users")
				{
					usersGroup.GET("", dbUserHandler.ListUsers)
					usersGroup.POST("", dbUserHandler.CreateUser)
				}
				databaseGroup.PATCH("/users/:id", dbUserHandler.UpdatePassword)
				databaseGroup.DELETE("/users/:id", dbUserHandler.DeleteUser)

				permissionsGroup := databaseGroup.Group("/permissions")
				{
					permissionsGroup.POST("", dbPermissionHandler.GrantPermission)
					permissionsGroup.DELETE("/:id", dbPermissionHandler.RevokePermission)
				}
				databaseGroup.GET("/users/:id/permissions", dbPermissionHandler.GetUserPermissions)

				queriesGroup := databaseGroup.Group("/instances/:id")
				{
					queriesGroup.POST("/query", dbQueryHandler.ExecuteQuery)
				}

				// T036-T037: 审计查询已迁移到 /api/v1/audit?subsystem=database
			}

			// ==================== Docker Management Subsystem ====================
			dockerGroup := protected.Group("/docker")
			dockerGroup.Use(middleware.RequireAdmin())
			{
				// Instance management
				instancesGroup := dockerGroup.Group("/instances")
				{
					instancesGroup.GET("", dockerInstanceHandler.GetInstances)
					instancesGroup.POST("", dockerInstanceHandler.CreateInstance)
					instancesGroup.GET("/:id", dockerInstanceHandler.GetInstance)
					instancesGroup.PUT("/:id", dockerInstanceHandler.UpdateInstance)
					instancesGroup.DELETE("/:id", dockerInstanceHandler.DeleteInstance)
					instancesGroup.POST("/:id/test-connection", dockerInstanceHandler.TestConnection)
				}

				// Container operations
				containersGroup := dockerGroup.Group("/instances/:id/containers")
				{
					containersGroup.GET("", dockerContainerHandler.GetContainers)
					containersGroup.GET("/:container_id", dockerContainerHandler.GetContainer)
					containersGroup.POST("/start", dockerContainerHandler.StartContainer)
					containersGroup.POST("/stop", dockerContainerHandler.StopContainer)
					containersGroup.POST("/restart", dockerContainerHandler.RestartContainer)
					containersGroup.POST("/pause", dockerContainerHandler.PauseContainer)
					containersGroup.POST("/unpause", dockerContainerHandler.UnpauseContainer)
					containersGroup.POST("/delete", dockerContainerHandler.DeleteContainer)

					// Container stats
					containersGroup.GET("/:container_id/stats", dockerStatsHandler.GetContainerStats)
					containersGroup.GET("/:container_id/stats/stream", dockerStatsHandler.GetContainerStatsStream)

					// Container logs
					containersGroup.GET("/:container_id/logs", dockerLogsHandler.GetContainerLogs)
					containersGroup.GET("/:container_id/logs/stream", dockerLogsHandler.GetContainerLogsStream)

					// Container terminal (T040)
					containersGroup.POST("/:container_id/terminal", dockerTerminalHandler.CreateTerminalSession)
				}

				// Image operations
				imagesGroup := dockerGroup.Group("/instances/:id/images")
				{
					imagesGroup.GET("", dockerImageHandler.GetImages)
					imagesGroup.GET("/:image_id", dockerImageHandler.GetImage)
					imagesGroup.POST("/delete", dockerImageHandler.DeleteImage)
					imagesGroup.POST("/tag", dockerImageHandler.TagImage)
					imagesGroup.POST("/pull", dockerImageHandler.PullImage)
				}

				// Audit logs
				dockerGroup.GET("/audit-logs", dockerAuditHandler.GetDockerAuditLogs)

				// Volume operations
				volumesGroup := dockerGroup.Group("/instances/:id/volumes")
				{
					volumesGroup.GET("", dockerVolumeHandler.GetVolumes)
					volumesGroup.GET("/:volume_name", dockerVolumeHandler.GetVolume)
					volumesGroup.POST("", dockerVolumeHandler.CreateVolume)
					volumesGroup.POST("/delete", dockerVolumeHandler.DeleteVolume)
					volumesGroup.POST("/prune", dockerVolumeHandler.PruneVolumes)
				}

				// Network operations
				networksGroup := dockerGroup.Group("/instances/:id/networks")
				{
					networksGroup.GET("", dockerNetworkHandler.GetNetworks)
					networksGroup.GET("/:network_id", dockerNetworkHandler.GetNetwork)
					networksGroup.POST("", dockerNetworkHandler.CreateNetwork)
					networksGroup.POST("/delete", dockerNetworkHandler.DeleteNetwork)
					networksGroup.POST("/connect", dockerNetworkHandler.ConnectNetwork)
					networksGroup.POST("/disconnect", dockerNetworkHandler.DisconnectNetwork)
				}

				// System operations
				systemGroup := dockerGroup.Group("/instances/:id/system")
				{
					systemGroup.GET("/info", dockerSystemHandler.GetSystemInfo)
					systemGroup.GET("/version", dockerSystemHandler.GetVersion)
					systemGroup.GET("/disk-usage", dockerSystemHandler.GetDiskUsage)
					systemGroup.GET("/ping", dockerSystemHandler.Ping)
					systemGroup.GET("/events/stream", dockerSystemHandler.GetEventsStream)
				}

				// Terminal recordings
				recordingsGroup := dockerGroup.Group("/recordings")
				{
					recordingsGroup.GET("", dockerRecordingHandler.GetRecordings)
					recordingsGroup.GET("/:id", dockerRecordingHandler.GetRecording)
					recordingsGroup.GET("/:id/playback", dockerRecordingHandler.GetRecordingPlayback)
					recordingsGroup.DELETE("/:id", dockerRecordingHandler.DeleteRecording)
					recordingsGroup.GET("/statistics", dockerRecordingHandler.GetRecordingStatistics)
				}
			}

			// Docker terminal WebSocket (T041 - outside protected group for token query param)
			router.GET("/api/v1/docker/terminal/:session_id", dockerTerminalHandler.HandleWebSocketTerminal)

			// ==================== MinIO Subsystem ====================
			minioGroup := protected.Group("/minio/instances/:id")
			{
				// Bucket operations
				minioGroup.GET("/buckets", minioBucketHandler.ListBuckets)
				minioGroup.POST("/buckets", minioBucketHandler.CreateBucket)
				minioGroup.GET("/buckets/:bucket", minioBucketHandler.GetBucket)
				minioGroup.PUT("/buckets/:bucket/policy", minioBucketHandler.UpdateBucketPolicy)
				minioGroup.DELETE("/buckets/:bucket", minioBucketHandler.DeleteBucket)

				// Object operations
				minioGroup.GET("/buckets/:bucket/objects", minioObjectHandler.ListObjects)
				minioGroup.GET("/buckets/:bucket/objects/:object", minioObjectHandler.GetObject)
				minioGroup.POST("/buckets/:bucket/objects", minioObjectHandler.UploadObject)
				minioGroup.DELETE("/buckets/:bucket/objects/:object", minioObjectHandler.DeleteObject)

				// File operations (generic)
				minioFileHandler := minio.NewFileHandler(*instanceRepo)
				minioGroup.GET("/files", minioFileHandler.List)
				minioGroup.POST("/files", minioFileHandler.Upload)
				minioGroup.GET("/files/download", minioFileHandler.DownloadURL)
				minioGroup.GET("/files/preview", minioFileHandler.PreviewURL)
				minioGroup.DELETE("/files", minioFileHandler.Delete)

				// User operations
				minioUserHandler := minio.NewUserHandler(*instanceRepo)
				minioGroup.GET("/users", minioUserHandler.ListUsers)
				minioGroup.POST("/users", minioUserHandler.CreateUser)
				minioGroup.DELETE("/users/:username", minioUserHandler.DeleteUser)
			}

			// ==================== Alert Management Subsystem ====================
			alertsGroup := protected.Group("/alerts")
			{
				// Alert rules
				alertsGroup.GET("/rules", alertHandler.ListAlertRules)
				alertsGroup.POST("/rules", alertHandler.CreateAlertRule)
				alertsGroup.GET("/rules/:rule_id", alertHandler.GetAlertRule)
				alertsGroup.PUT("/rules/:rule_id", alertHandler.UpdateAlertRule)
				alertsGroup.DELETE("/rules/:rule_id", alertHandler.DeleteAlertRule)
				alertsGroup.POST("/rules/:rule_id/toggle", alertHandler.ToggleAlertRule)

				// Alert events
				alertsGroup.GET("/events", alertHandler.ListAlertEvents)
				alertsGroup.GET("/events/active", alertHandler.GetActiveAlertEvents)
				alertsGroup.GET("/events/:event_id", alertHandler.GetAlertEvent)
				alertsGroup.POST("/events/:event_id/acknowledge", alertHandler.AcknowledgeAlertEvent)
				alertsGroup.POST("/events/:event_id/resolve", alertHandler.ResolveAlertEvent)

				// Statistics
				alertsGroup.GET("/statistics", alertHandler.GetAlertStatistics)
			}

			// ==================== Scheduler Subsystem ====================
			// T022: Scheduler API endpoints
			schedulerGroup := protected.Group("/scheduler")
			{
				// Task management
				schedulerGroup.GET("/tasks", schedulerTaskHandler.ListTasks)
				schedulerGroup.GET("/tasks/:id", schedulerTaskHandler.GetTask)
				schedulerGroup.POST("/tasks/:id/enable", schedulerTaskHandler.EnableTask)
				schedulerGroup.POST("/tasks/:id/disable", schedulerTaskHandler.DisableTask)
				schedulerGroup.POST("/tasks/:id/trigger", schedulerTaskHandler.TriggerTask)

				// Execution history
				schedulerGroup.GET("/executions", schedulerExecutionHandler.ListExecutions)
				schedulerGroup.GET("/executions/:id", schedulerExecutionHandler.GetExecution)

				// Statistics
				schedulerGroup.GET("/stats", schedulerStatsHandler.GetStats)
			}

			// ==================== Audit Log Subsystem ====================
			auditGroup := protected.Group("/audit")
			{
				// Main audit log endpoints (legacy)
				auditGroup.GET("", auditHandler.ListAuditLogs)
				auditGroup.GET("/:log_id", auditHandler.GetAuditLog)
				auditGroup.GET("/recent", auditHandler.ListRecentLogs)
				auditGroup.GET("/failed", auditHandler.ListFailedActions)

				// Analytics (legacy)
				auditGroup.GET("/timeline", auditHandler.GetActivityTimeline)
				auditGroup.GET("/statistics", auditHandler.GetAuditStatistics)
				auditGroup.GET("/actions", auditHandler.GetDistinctActions)
				auditGroup.GET("/resource-types", auditHandler.GetDistinctResourceTypes)
				auditGroup.GET("/search", auditHandler.SearchAuditLogs)

				// Resource-specific (legacy)
				auditGroup.GET("/resources/:resource_type/:resource_id", auditHandler.ListResourceAuditLogs)

				// ==================== New Unified Audit API (T023) ====================
				// Unified audit events
				auditGroup.GET("/events", auditEventHandler.ListEvents)
				auditGroup.GET("/events/:id", auditEventHandler.GetEvent)

				// Audit configuration
				auditGroup.GET("/config", auditConfigHandler.GetConfig)
				auditGroup.PUT("/config", auditConfigHandler.UpdateConfig)
			}

			// ==================== User Management Subsystem ====================
			usersGroup := protected.Group("/users")
			{
				usersGroup.GET("", userHandler.ListUsers)
				usersGroup.POST("", userHandler.CreateUser)
				usersGroup.GET("/:user_id", userHandler.GetUser)
				usersGroup.PATCH("/:user_id", userHandler.UpdateUser)
				usersGroup.DELETE("/:user_id", userHandler.DeleteUser)

				// User roles management
				usersGroup.GET("/:user_id/roles", userHandler.GetUserRoles)
				usersGroup.POST("/:user_id/roles", userHandler.AssignRole)
				usersGroup.DELETE("/:user_id/roles/:role_id", userHandler.RevokeRole)
			}

			// User-specific audit logs
			protected.GET("/users/:user_id/audit", auditHandler.ListUserAuditLogs)

			// ==================== VMs (Host Monitoring) Subsystem ====================
			vmsGroup := protected.Group("/vms")
			{
				// Host node management
				hostsGroup := vmsGroup.Group("/hosts")
				{
					hostsGroup.POST("", hostHandler.CreateHost)
					hostsGroup.GET("", hostHandler.ListHosts)
					hostsGroup.GET("/:id", hostHandler.GetHost)
					hostsGroup.PUT("/:id", hostHandler.UpdateHost)
					hostsGroup.DELETE("/:id", hostHandler.DeleteHost)

					// Agent management
					hostsGroup.POST("/:id/regenerate-secret-key", hostHandler.RegenerateSecretKey)
					hostsGroup.GET("/:id/agent-install-command", hostHandler.GetAgentInstallCommand)

					// Host state queries
					hostsGroup.GET("/:id/state/current", hostHandler.GetCurrentState)
					hostsGroup.GET("/:id/state/history", hostHandler.GetHistoryState)

					// T038: Host activities API 已移除，使用统一审计 API
					// GET /api/v1/audit/events?subsystem=host&resource.identifier=<host_id>
					// POST 不再支持（审计由系统自动记录）

					// Service probe history (for multi-line chart)
					hostsGroup.GET("/:id/probe-history", serviceMonitorHandler.GetHostProbeHistory)
				}

				// MinIO Permission routes (global under /minio)
				minioPermHandler := minio.NewPermissionHandler(*instanceRepo)
				minioAPI := protected.Group("/minio")
				{
					// MinIO instances CRUD
					minioInstHandler := minio.NewMinioInstanceHandler(*instanceRepo)
					minioAPI.POST("/instances", minioInstHandler.Create)
					minioAPI.GET("/instances", minioInstHandler.List)
					minioAPI.GET("/instances/:id", minioInstHandler.Get)
					minioAPI.PUT("/instances/:id", minioInstHandler.Update)
					minioAPI.DELETE("/instances/:id", minioInstHandler.Delete)
					minioAPI.POST("/instances/:id/test", minioInstHandler.Test)

					minioAPI.POST("/permissions", minioPermHandler.GrantPermission)
					minioAPI.GET("/permissions", minioPermHandler.ListPermissions)
					minioAPI.DELETE("/permissions/:id", minioPermHandler.RevokePermission)

					// Shares
					minioShareHandler := minio.NewShareHandler(*instanceRepo)
					minioAPI.POST("/shares", minioShareHandler.CreateShare)
					minioAPI.GET("/shares", minioShareHandler.ListShares)
					minioAPI.DELETE("/shares/:id", minioShareHandler.RevokeShare)
				}

				// Host groups (simplified - just list unique group names)
				hostGroupsGroup := vmsGroup.Group("/host-groups")
				{
					hostGroupsGroup.GET("", hostGroupHandler.ListGroups)
				}

				// Service monitoring
				serviceMonitorsGroup := vmsGroup.Group("/service-monitors")
				{
					serviceMonitorsGroup.GET("", serviceMonitorHandler.ListMonitors)
					serviceMonitorsGroup.GET("/overview", serviceMonitorHandler.GetOverview)
					serviceMonitorsGroup.GET("/topology", serviceMonitorHandler.GetNetworkTopology)
					serviceMonitorsGroup.POST("", serviceMonitorHandler.CreateMonitor)
					serviceMonitorsGroup.GET("/:id", serviceMonitorHandler.GetMonitor)
					serviceMonitorsGroup.PUT("/:id", serviceMonitorHandler.UpdateMonitor)
					serviceMonitorsGroup.DELETE("/:id", serviceMonitorHandler.DeleteMonitor)
					serviceMonitorsGroup.POST("/:id/trigger", serviceMonitorHandler.TriggerProbe)
					serviceMonitorsGroup.GET("/:id/availability", serviceMonitorHandler.GetAvailability)
					serviceMonitorsGroup.GET("/:id/history", serviceMonitorHandler.GetProbeHistory)
				}

				// WebSSH
				websshGroup := vmsGroup.Group("/webssh")
				{
					websshGroup.POST("/sessions", websshHandler.CreateSession)
					websshGroup.GET("/sessions", websshHandler.ListSessions)
					websshGroup.GET("/sessions/all", websshHandler.ListAllSessions) // All sessions with pagination
					websshGroup.GET("/sessions/:session_id", websshHandler.GetSessionDetail)
					websshGroup.GET("/sessions/:session_id/playback", websshHandler.GetRecording)
					websshGroup.DELETE("/sessions/:session_id", websshHandler.CloseSession)
					websshGroup.GET("/:session_id", websshHandler.HandleWebSocket)
				}

				// Monitor alert rules and events
				alertRulesGroup := vmsGroup.Group("/alert-rules")
				{
					alertRulesGroup.POST("", monitorAlertHandler.CreateRule)
					alertRulesGroup.GET("", monitorAlertHandler.ListRules)
					alertRulesGroup.GET("/:id", monitorAlertHandler.GetRule)
					alertRulesGroup.PUT("/:id", monitorAlertHandler.UpdateRule)
					alertRulesGroup.DELETE("/:id", monitorAlertHandler.DeleteRule)
				}

				alertEventsGroup := vmsGroup.Group("/alert-events")
				{
					alertEventsGroup.GET("", monitorAlertHandler.ListEvents)
					alertEventsGroup.POST("/:id/acknowledge", monitorAlertHandler.AcknowledgeEvent)
					alertEventsGroup.POST("/:id/resolve", monitorAlertHandler.ResolveEvent)
				}

				// WebSocket real-time monitoring
				wsGroup := vmsGroup.Group("/ws")
				{
					wsGroup.GET("/hosts/monitor", websocketHandler.HostMonitor)
					wsGroup.GET("/service-probes", websocketHandler.ServiceProbe)
					wsGroup.GET("/alert-events", websocketHandler.AlertEvents)
				}
			}
		}
	}
}
