package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/auth"
)

type createPasswordUser struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name"`
}

func CreateSuperUser(c *gin.Context) {
	var userreq createPasswordUser
	if err := c.ShouldBindJSON(&userreq); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	userRepo := repository.NewUserRepository(models.DB)
	ctx := context.Background()

	uc, err := userRepo.Count(ctx)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to count users"})
		return
	}

	if uc > 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "super user already exists"})
		return
	}

	// Hash the password before storing
	passwordHasher := auth.NewPasswordHasher()
	hashedPassword, err := passwordHasher.Hash(userreq.Password)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to hash password"})
		return
	}

	user := &models.User{
		Username: userreq.Username,
		Password: hashedPassword,
		Name:     userreq.Name,
		FullName: userreq.Name,
		Provider: "password",
		Enabled:  true,
		IsAdmin:  true, // Super user should be admin
	}

	// Create user
	if err := userRepo.Create(ctx, user); err != nil {
		c.JSON(500, gin.H{"error": "failed to create super user"})
		return
	}

	c.JSON(201, user)
}

func CreatePasswordUser(c *gin.Context) {
	var userreq createPasswordUser
	if err := c.ShouldBindJSON(&userreq); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	userRepo := repository.NewUserRepository(models.DB)
	ctx := context.Background()

	// Hash the password before storing
	passwordHasher := auth.NewPasswordHasher()
	hashedPassword, err := passwordHasher.Hash(userreq.Password)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to hash password"})
		return
	}

	user := &models.User{
		Username: userreq.Username,
		Password: hashedPassword,
		Name:     userreq.Name,
		FullName: userreq.Name,
		Provider: "password",
		Enabled:  true,
	}

	_, err = userRepo.GetByUsername(ctx, user.Username)
	if err == nil {
		c.JSON(400, gin.H{"error": "user already exists"})
		return
	}

	if err := userRepo.Create(ctx, user); err != nil {
		c.JSON(500, gin.H{"error": "failed to create user"})
		return
	}
	c.JSON(201, user)
}

func ListUsers(c *gin.Context) {
	page := 1
	size := 20
	if p := c.Query("page"); p != "" {
		_, _ = fmt.Sscanf(p, "%d", &page)
		if page <= 0 {
			page = 1
		}
	}
	if s := c.Query("size"); s != "" {
		_, _ = fmt.Sscanf(s, "%d", &size)
		if size <= 0 {
			size = 20
		}
	}

	userRepo := repository.NewUserRepository(models.DB)
	ctx := context.Background()

	filter := &repository.ListUsersFilter{
		Page:     page,
		PageSize: size,
	}
	users, total, err := userRepo.ListUsers(ctx, filter)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to list users"})
		return
	}
	// Simplified: removed RBAC system
	c.JSON(200, gin.H{"users": users, "total": total, "page": page, "size": size})
}

func UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	userRepo := repository.NewUserRepository(models.DB)
	ctx := context.Background()

	user, err := userRepo.GetByID(ctx, id)
	if err != nil {
		c.JSON(404, gin.H{"error": "user not found"})
		return
	}
	if req.Name != "" {
		user.Name = req.Name
		user.FullName = req.Name
	}
	if req.AvatarURL != "" {
		user.AvatarURL = req.AvatarURL
	}

	if err := userRepo.Update(ctx, user); err != nil {
		c.JSON(500, gin.H{"error": "failed to update user"})
		return
	}
	c.JSON(200, user)
}

func DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	userRepo := repository.NewUserRepository(models.DB)
	ctx := context.Background()

	if err := userRepo.Delete(ctx, id); err != nil {
		c.JSON(500, gin.H{"error": "failed to delete user"})
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func ResetPassword(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	userRepo := repository.NewUserRepository(models.DB)
	ctx := context.Background()

	// Hash the password
	passwordHasher := auth.NewPasswordHasher()
	hashedPassword, err := passwordHasher.Hash(req.Password)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to hash password"})
		return
	}

	// Update password
	updates := map[string]interface{}{
		"password": hashedPassword,
	}
	if err := userRepo.UpdateFields(ctx, id, updates); err != nil {
		c.JSON(500, gin.H{"error": "failed to reset password"})
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func SetUserEnabled(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	userRepo := repository.NewUserRepository(models.DB)
	ctx := context.Background()

	if err := userRepo.SetEnabled(ctx, id, req.Enabled); err != nil {
		c.JSON(500, gin.H{"error": "failed to set enabled"})
		return
	}
	c.JSON(200, gin.H{"success": true})
}
