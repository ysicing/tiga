package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// ClusterContextKey is the key for storing cluster in gin.Context
const ClusterContextKey = "cluster"

// ClusterContext middleware extracts cluster ID from URL parameter and loads cluster into context
// This middleware should be used for routes that require cluster context: /api/v1/k8s/clusters/:id/*
func ClusterContext(clusterRepo repository.ClusterRepositoryInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Extract cluster ID from URL parameter
		idStr := c.Param("id")
		if idStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": "Cluster ID is required",
			})
			c.Abort()
			return
		}

		// Parse UUID
		clusterID, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": "Invalid cluster ID format",
			})
			c.Abort()
			return
		}

		// Load cluster from database
		cluster, err := clusterRepo.GetByID(ctx, clusterID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{
					"code":    http.StatusNotFound,
					"message": "Cluster not found",
				})
				c.Abort()
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": "Failed to load cluster",
			})
			c.Abort()
			return
		}

		// Store cluster in context for downstream handlers
		c.Set(ClusterContextKey, cluster)

		c.Next()
	}
}

// GetClusterFromContext extracts cluster from gin.Context
// Returns cluster and true if found, nil and false otherwise
func GetClusterFromContext(c *gin.Context) (*models.Cluster, bool) {
	value, exists := c.Get(ClusterContextKey)
	if !exists {
		return nil, false
	}
	cluster, ok := value.(*models.Cluster)
	return cluster, ok
}

// MustGetClusterFromContext extracts cluster from gin.Context
// Panics if cluster is not found (should only be used after ClusterContext middleware)
func MustGetClusterFromContext(c *gin.Context) *models.Cluster {
	cluster, ok := GetClusterFromContext(c)
	if !ok {
		panic("cluster not found in context - did you forget to use ClusterContext middleware?")
	}
	return cluster
}
