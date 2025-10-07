package api

import (
	"github.com/gin-gonic/gin"
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

	installhandlers "github.com/ysicing/tiga/internal/install/handlers"
	alertservices "github.com/ysicing/tiga/internal/services/alert"
	authservices "github.com/ysicing/tiga/internal/services/auth"
	hostservices "github.com/ysicing/tiga/internal/services/host"
	monitorservices "github.com/ysicing/tiga/internal/services/monitor"
	websshservices "github.com/ysicing/tiga/internal/services/webssh"
	pkghandlers "github.com/ysicing/tiga/pkg/handlers"
	pkgmiddleware "github.com/ysicing/tiga/pkg/middleware"
)

// SetupRoutes configures all application routes
func SetupRoutes(router *gin.Engine, db *gorm.DB, configPath string, jwtManager *authservices.JWTManager, jwtSecret string) {
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

	// Host monitoring repositories
	hostRepo := repository.NewHostRepository(db)
	serviceRepo := repository.NewServiceRepository(db)
	monitorAlertRepo := repository.NewMonitorAlertRepository(db)

	// Initialize session and login services using the shared jwtManager
	sessionService := authservices.NewSessionService(db)
	loginService := authservices.NewLoginService(db, jwtManager, sessionService)

	// Initialize services
	instanceService := services.NewInstanceService(instanceRepo)

	// Host monitoring services
	serverURL := "http://localhost:12306" // TODO: Get from config
	agentManager := hostservices.NewAgentManager(hostRepo, db)
	stateCollector := hostservices.NewStateCollector(hostRepo, agentManager)
	hostService := hostservices.NewHostService(hostRepo, agentManager, stateCollector, serverURL)
	probeScheduler := monitorservices.NewServiceProbeScheduler(serviceRepo)
	probeService := monitorservices.NewServiceProbeService(serviceRepo, probeScheduler)
	sessionManager := websshservices.NewSessionManager(db)
	terminalManager := hostservices.NewTerminalManager()

	// Start background services
	_ = alertservices.NewAlertEngine(monitorAlertRepo, hostRepo) // Runs in background
	expiryScheduler := hostservices.NewExpiryScheduler(hostRepo, monitorAlertRepo, db)
	expiryScheduler.Start()

	// Initialize handlers
	instanceHandler := handlers.NewInstanceHandler(instanceRepo)
	healthHandler := instances.NewHealthHandler(instanceService)
	metricsHandler := instances.NewMetricsHandler(instanceService)
	alertHandler := handlers.NewAlertHandler(alertRepo)
	auditHandler := handlers.NewAuditLogHandler(auditRepo)

	// Host monitoring handlers
	hostHandler := handlers.NewHostHandler(hostService)
	hostGroupHandler := handlers.NewHostGroupHandler(db)
	serviceMonitorHandler := handlers.NewServiceMonitorHandler(probeService)
	// TODO: Create monitor_alert_handler.go for MonitorAlertRule/MonitorAlertEvent
	websshHandler := handlers.NewWebSSHHandler(sessionManager, terminalManager, agentManager)
	websocketHandler := handlers.NewWebSocketHandler(stateCollector)

	// MinIO handlers
	minioBucketHandler := minio.NewBucketHandler(*instanceRepo)
	minioObjectHandler := minio.NewObjectHandler(*instanceRepo)

	// Auth handler for /api/auth routes - using new system
	authHandler := handlers.NewAuthHandler(loginService, sessionService, nil)

	// ==================== Auth Routes (No Auth Required /api/auth) ====================
	// Note: These are at /api/auth (not /api/v1/auth) for compatibility with frontend
	authGroup := router.Group("/api/auth")
	{
		// Get available OAuth providers (no auth required)
		authGroup.GET("/providers", authHandler.GetOAuthProviders)
		// Password login endpoint - using new system
		authGroup.POST("/login/password", authHandler.Login)
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
	}

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		// ==================== Setup & Initialization (No Auth Required) ====================
		// Initialization check endpoint (no auth required)
		v1.GET("/init_check", pkghandlers.NewInitCheckHandler(configPath))

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
		protected.Use(middleware.AuthRequired())
		{
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

			// ==================== Instance Management Subsystem ====================
			instancesGroup := protected.Group("/instances")
			{
				// Instance CRUD
				instancesGroup.GET("", instanceHandler.ListInstances)
				instancesGroup.POST("", instanceHandler.CreateInstance)
				instancesGroup.GET("/:instance_id", instanceHandler.GetInstance)
				instancesGroup.PUT("/:instance_id", instanceHandler.UpdateInstance)
				instancesGroup.DELETE("/:instance_id", instanceHandler.DeleteInstance)

				// Instance health and metrics
				instancesGroup.GET("/:instance_id/health", healthHandler.GetInstanceHealth)
				instancesGroup.GET("/:instance_id/metrics", metricsHandler.GetInstanceMetrics)
			}

			// ==================== MinIO Subsystem ====================
			minioGroup := protected.Group("/minio/instances/:id")
			{
				// Bucket operations
				minioGroup.GET("/buckets", minioBucketHandler.ListBuckets)
				minioGroup.POST("/buckets", minioBucketHandler.CreateBucket)
				minioGroup.DELETE("/buckets/:bucket", minioBucketHandler.DeleteBucket)

				// Object operations
				minioGroup.GET("/buckets/:bucket/objects", minioObjectHandler.ListObjects)
				minioGroup.GET("/buckets/:bucket/objects/:object", minioObjectHandler.GetObject)
				minioGroup.POST("/buckets/:bucket/objects", minioObjectHandler.UploadObject)
				minioGroup.DELETE("/buckets/:bucket/objects/:object", minioObjectHandler.DeleteObject)
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

			// ==================== Audit Log Subsystem ====================
			auditGroup := protected.Group("/audit")
			{
				// Main audit log endpoints
				auditGroup.GET("", auditHandler.ListAuditLogs)
				auditGroup.GET("/:log_id", auditHandler.GetAuditLog)
				auditGroup.GET("/recent", auditHandler.ListRecentLogs)
				auditGroup.GET("/failed", auditHandler.ListFailedActions)

				// Analytics
				auditGroup.GET("/timeline", auditHandler.GetActivityTimeline)
				auditGroup.GET("/statistics", auditHandler.GetAuditStatistics)
				auditGroup.GET("/actions", auditHandler.GetDistinctActions)
				auditGroup.GET("/resource-types", auditHandler.GetDistinctResourceTypes)
				auditGroup.GET("/search", auditHandler.SearchAuditLogs)

				// Resource-specific
				auditGroup.GET("/resources/:resource_type/:resource_id", auditHandler.ListResourceAuditLogs)
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

					// Host state queries
					hostsGroup.GET("/:id/state/current", hostHandler.GetCurrentState)
					hostsGroup.GET("/:id/state/history", hostHandler.GetHistoryState)
				}

				// Host groups
				hostGroupsGroup := vmsGroup.Group("/host-groups")
				{
					hostGroupsGroup.POST("", hostGroupHandler.CreateGroup)
					hostGroupsGroup.GET("", hostGroupHandler.ListGroups)
					hostGroupsGroup.DELETE("/:id", hostGroupHandler.DeleteGroup)
					hostGroupsGroup.POST("/:id/hosts", hostGroupHandler.AddHosts)
					hostGroupsGroup.DELETE("/:id/hosts/:host_id", hostGroupHandler.RemoveHost)
				}

				// Service monitoring
				serviceMonitorsGroup := vmsGroup.Group("/service-monitors")
				{
					serviceMonitorsGroup.POST("", serviceMonitorHandler.CreateMonitor)
					serviceMonitorsGroup.GET("/:id", serviceMonitorHandler.GetMonitor)
					serviceMonitorsGroup.PUT("/:id", serviceMonitorHandler.UpdateMonitor)
					serviceMonitorsGroup.DELETE("/:id", serviceMonitorHandler.DeleteMonitor)
					serviceMonitorsGroup.POST("/:id/trigger", serviceMonitorHandler.TriggerProbe)
					serviceMonitorsGroup.GET("/:id/availability", serviceMonitorHandler.GetAvailability)
				}

				// WebSSH
				websshGroup := vmsGroup.Group("/webssh")
				{
					websshGroup.POST("/sessions", websshHandler.CreateSession)
					websshGroup.GET("/sessions", websshHandler.ListSessions)
					websshGroup.DELETE("/sessions/:session_id", websshHandler.CloseSession)
					websshGroup.GET("/:session_id", websshHandler.HandleWebSocket)
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
