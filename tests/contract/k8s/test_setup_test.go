package k8s_test

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/api/handlers/cluster"
	"github.com/ysicing/tiga/internal/config"
	"github.com/ysicing/tiga/internal/db"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// setupTestAPI creates a test router with K8s cluster routes
func setupTestAPI(t *testing.T) (*gin.Engine, *gorm.DB, func()) {
	// Create in-memory SQLite database for testing
	cfg := &config.DatabaseConfig{
		Type: "sqlite",
		Name: ":memory:",
	}

	database, err := db.NewDatabase(cfg)
	require.NoError(t, err, "Failed to create test database")

	// Run migrations
	err = database.AutoMigrate()
	require.NoError(t, err, "Failed to run migrations")

	// Create test admin user
	adminUser := &models.User{
		Username: "testadmin",
		Email:    "admin@test.com",
		Password: "hashedpassword",
	}
	err = database.DB.Create(adminUser).Error
	require.NoError(t, err, "Failed to create admin user")

	// Setup repositories
	clusterRepo := repository.NewClusterRepository(database.DB)
	resourceHistoryRepo := repository.NewResourceHistoryRepository(database.DB)

	// Setup handler
	appConfig := &config.Config{
		Kubernetes: config.KubernetesConfig{
			NodeTerminalImage: "alpine:latest",
		},
		Prometheus: config.PrometheusConfig{
			AutoDiscovery:    true,
			DiscoveryTimeout: 30,
		},
	}
	clusterHandler := cluster.NewClusterHandler(clusterRepo, resourceHistoryRepo, appConfig)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Register K8s cluster routes (matching internal/api/routes.go structure)
	apiGroup := router.Group("/api/v1")
	{
		k8sGroup := apiGroup.Group("/k8s")
		{
			clustersGroup := k8sGroup.Group("/clusters")
			{
				// Cluster CRUD operations
				clustersGroup.GET("", clusterHandler.List)
				clustersGroup.POST("", clusterHandler.Create)
				clustersGroup.GET("/:id", clusterHandler.Get)
				clustersGroup.PUT("/:id", clusterHandler.Update)
				clustersGroup.DELETE("/:id", clusterHandler.Delete)

				// Cluster operations
				clustersGroup.POST("/:id/test-connection", clusterHandler.TestConnection)
				clustersGroup.POST("/:id/set-default", clusterHandler.SetDefault)

				// Prometheus discovery (Phase 1)
				clustersGroup.POST("/:id/prometheus/rediscover", clusterHandler.RediscoverPrometheus)

				// CRD detection (Phase 2)
				clustersGroup.GET("/:id/crds", clusterHandler.DetectCRDs)

				// CloneSet operations (Phase 2)
				clustersGroup.GET("/:id/clonesets", clusterHandler.ListCloneSets)
				clustersGroup.PUT("/:id/clonesets/:name/scale", clusterHandler.ScaleCloneSet)
				clustersGroup.POST("/:id/clonesets/:name/restart", clusterHandler.RestartCloneSet)
			}
		}
	}

	cleanup := func() {
		sqlDB, _ := database.DB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return router, database.DB, cleanup
}
