package minio

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "gorm.io/gorm"

    "github.com/ysicing/tiga/internal/models"
)

type UserRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) Create(ctx context.Context, user *models.MinIOUser) error {
    if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
        return fmt.Errorf("create minio user: %w", err)
    }
    return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.MinIOUser, error) {
    var u models.MinIOUser
    if err := r.db.WithContext(ctx).First(&u, "id = ?", id).Error; err != nil {
        return nil, err
    }
    return &u, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, instanceID uuid.UUID, username string) (*models.MinIOUser, error) {
    var u models.MinIOUser
    if err := r.db.WithContext(ctx).Where("instance_id = ? AND username = ?", instanceID, username).First(&u).Error; err != nil {
        return nil, err
    }
    return &u, nil
}

func (r *UserRepository) ListByInstance(ctx context.Context, instanceID uuid.UUID) ([]*models.MinIOUser, error) {
    var users []*models.MinIOUser
    if err := r.db.WithContext(ctx).Where("instance_id = ?", instanceID).Find(&users).Error; err != nil {
        return nil, err
    }
    return users, nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
    res := r.db.WithContext(ctx).Delete(&models.MinIOUser{}, id)
    if res.Error != nil { return res.Error }
    if res.RowsAffected == 0 { return gorm.ErrRecordNotFound }
    return nil
}

