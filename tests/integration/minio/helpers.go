package minio_integration

import (
    "context"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/stretchr/testify/require"
    "gorm.io/gorm"

    apiHandlers "github.com/ysicing/tiga/internal/api/handlers/minio"
    "github.com/ysicing/tiga/internal/config"
    "github.com/ysicing/tiga/internal/db"
    "github.com/ysicing/tiga/internal/models"
    "github.com/ysicing/tiga/internal/repository"
    tcminio "github.com/ysicing/tiga/tests/testcontainers"
)

type TestEnv struct {
    Router      *gin.Engine
    DB          *gorm.DB
    Server      *httptest.Server
    InstanceID  uuid.UUID
    Minio       *tcminio.MinioContainer
    CleanupFunc func()
}

// SetupMinioTestEnv creates a minimal gin router with only MinIO routes and a sqlite DB.
func SetupMinioTestEnv(t *testing.T) *TestEnv {
    t.Helper()
    ctx := context.Background()

    // Start MinIO container
    mc, err := tcminio.StartMinioContainer(ctx)
    if err != nil {
        t.Skipf("skipping MinIO contract test: failed to start container: %v", err)
        return &TestEnv{CleanupFunc: func() {}}
    }

    // Setup in-memory DB
    cfg := &config.DatabaseConfig{Type: "sqlite", Name: ":memory:"}
    database, err := db.NewDatabase(cfg)
    require.NoError(t, err)
    require.NoError(t, database.AutoMigrate())

    // Create owner user (required by Instance model)
    owner := &models.User{Username: "tester", Email: "tester@example.com", Password: "dummy"}
    require.NoError(t, database.DB.Create(owner).Error)

    // Create MinIO instance record
    instance := &models.Instance{
        Name:   "minio-test",
        Type:   "minio",
        OwnerID: owner.ID,
        Connection: models.JSONB{
            "host": mc.Host,
            "port": mc.Port,
        },
        Config: models.JSONB{
            "access_key": mc.AccessKey,
            "secret_key": mc.SecretKey,
            "use_ssl":    false,
        },
        Status: "running",
        Health: "healthy",
    }
    require.NoError(t, database.DB.Create(instance).Error)

    // Minimal router with only required MinIO routes (no auth middleware in tests)
    gin.SetMode(gin.TestMode)
    r := gin.New()
    instRepo := repository.NewInstanceRepository(database.DB)

    bucketHandler := apiHandlers.NewBucketHandler(*instRepo)
    objectHandler := apiHandlers.NewObjectHandler(*instRepo)

    group := r.Group("/api/v1/minio/instances/:id")
    {
        group.GET("/buckets", bucketHandler.ListBuckets)
        group.POST("/buckets", bucketHandler.CreateBucket)
        group.GET("/buckets/:bucket", bucketHandler.GetBucket)
        group.PUT("/buckets/:bucket/policy", bucketHandler.UpdateBucketPolicy)
        group.DELETE("/buckets/:bucket", bucketHandler.DeleteBucket)

        group.GET("/buckets/:bucket/objects", objectHandler.ListObjects)
        group.GET("/buckets/:bucket/objects/:object", objectHandler.GetObject)
        group.POST("/buckets/:bucket/objects", objectHandler.UploadObject)
        group.DELETE("/buckets/:bucket/objects/:object", objectHandler.DeleteObject)
    }

    server := httptest.NewServer(r)

    cleanup := func() {
        server.Close()
        _ = database.Close()
        _ = mc.Terminate(ctx)
    }

    return &TestEnv{
        Router:      r,
        DB:          database.DB,
        Server:      server,
        InstanceID:  instance.ID,
        Minio:       mc,
        CleanupFunc: cleanup,
    }
}
