package minio

import (
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	mrepo "github.com/ysicing/tiga/internal/repository/minio"
	msvc "github.com/ysicing/tiga/internal/services/minio"
)

// FileHandler handles file operations under /api/v1/minio/instances/:id/files
type FileHandler struct {
	instanceRepo repository.InstanceRepository
}

func NewFileHandler(instanceRepo repository.InstanceRepository) *FileHandler {
	return &FileHandler{instanceRepo: instanceRepo}
}

// List files
// GET /api/v1/minio/instances/:id/files?bucket=...&prefix=...
func (h *FileHandler) List(c *gin.Context) {
	instanceID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	bucket := c.Query("bucket")
	if bucket == "" {
		handlers.RespondBadRequest(c, fmt.Errorf("bucket is required"))
		return
	}
	prefix := c.DefaultQuery("prefix", "")

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}
	if instance.Type != "minio" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not MinIO type"))
		return
	}

	svc := msvc.NewFileService(&h.instanceRepo)
	objs, err := svc.ListObjects(c.Request.Context(), instance.ID, bucket, prefix, true)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	var items []map[string]interface{}
	for _, o := range objs {
		items = append(items, map[string]interface{}{"key": o.Key, "size": o.Size, "etag": o.ETag, "last_modified": o.LastModified})
	}
	handlers.RespondSuccess(c, gin.H{"bucket": bucket, "prefix": prefix, "objects": items})
}

// Upload file
// POST /api/v1/minio/instances/:id/files (multipart form: bucket, name, file)
func (h *FileHandler) Upload(c *gin.Context) {
	instanceID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	bucket := c.PostForm("bucket")
	name := c.PostForm("name")
	if bucket == "" || name == "" {
		handlers.RespondBadRequest(c, fmt.Errorf("bucket and name are required"))
		return
	}
	// Path traversal / invalid key guard
	if strings.HasPrefix(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		handlers.RespondBadRequest(c, fmt.Errorf("invalid object key"))
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}
	defer func(f multipart.File) { _ = f.Close() }(file)

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}
	if instance.Type != "minio" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not MinIO type"))
		return
	}

	svc := msvc.NewFileService(&h.instanceRepo)
	info, err := svc.Upload(c.Request.Context(), instance.ID, bucket, name, file, header.Size, header.Header.Get("Content-Type"))
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	// Audit upload
	db := getDB(c)
	logger := msvc.NewAuditLogger(mrepo.NewAuditRepository(db))
	_ = logger.LogOperation(c.Request.Context(), instance.ID, "file", "object", name, "upload", "success", "", nil, "", c.ClientIP(), models.JSONB{"bucket": bucket, "size": info.Size})

	handlers.RespondCreated(c, gin.H{"bucket": bucket, "key": name, "size": info.Size, "etag": info.ETag})
}

// Presigned download URL (7 days)
// GET /api/v1/minio/instances/:id/files/download?bucket=...&key=...
func (h *FileHandler) DownloadURL(c *gin.Context) {
	h.presignURL(c, 7*24*time.Hour)
}

// Presigned preview URL (15 minutes)
// GET /api/v1/minio/instances/:id/files/preview?bucket=...&key=...
func (h *FileHandler) PreviewURL(c *gin.Context) {
	h.presignURL(c, 15*time.Minute)
}

func (h *FileHandler) presignURL(c *gin.Context, expiry time.Duration) {
	instanceID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}
	bucket := c.Query("bucket")
	key := c.Query("key")
	if bucket == "" || key == "" {
		handlers.RespondBadRequest(c, fmt.Errorf("bucket and key are required"))
		return
	}
	if strings.HasPrefix(key, "/") || strings.Contains(key, "\\") || strings.Contains(key, "..") {
		handlers.RespondBadRequest(c, fmt.Errorf("invalid object key"))
		return
	}

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}
	if instance.Type != "minio" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not MinIO type"))
		return
	}

	svc := msvc.NewFileService(&h.instanceRepo)
	u, err := svc.PresignedDownload(c.Request.Context(), instance.ID, bucket, key, expiry)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	handlers.RespondSuccess(c, gin.H{"url": u})
}

// Delete files
// DELETE /api/v1/minio/instances/:id/files  { "bucket": ..., "keys": [ ... ] }
func (h *FileHandler) Delete(c *gin.Context) {
	instanceID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}
	var req struct {
		Bucket string   `json:"bucket"`
		Keys   []string `json:"keys"`
	}
	if !handlers.BindJSON(c, &req) {
		return
	}
	if req.Bucket == "" || len(req.Keys) == 0 {
		handlers.RespondBadRequest(c, fmt.Errorf("bucket and keys are required"))
		return
	}
	for _, k := range req.Keys {
		if strings.HasPrefix(k, "/") || strings.Contains(k, "\\") || strings.Contains(k, "..") {
			handlers.RespondBadRequest(c, fmt.Errorf("invalid object key"))
			return
		}
	}

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}
	if instance.Type != "minio" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not MinIO type"))
		return
	}

	svc := msvc.NewFileService(&h.instanceRepo)
	if err := svc.DeleteBatch(c.Request.Context(), instance.ID, req.Bucket, req.Keys); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	// Audit delete
	db := getDB(c)
	logger := msvc.NewAuditLogger(mrepo.NewAuditRepository(db))
	_ = logger.LogOperation(c.Request.Context(), instance.ID, "file", "object", "batch", "delete", "success", "", nil, "", c.ClientIP(), models.JSONB{"bucket": req.Bucket, "count": len(req.Keys)})
	handlers.RespondSuccess(c, gin.H{"deleted": len(req.Keys)})
}
