package scheduler

import (
	"context"

	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// TaskRepository defines the interface for scheduled task data access
// T022: Repository for ScheduledTask CRUD operations
//
// Reference: .claude/specs/006-gitness-tiga/tasks.md T022
type TaskRepository interface {
	Create(ctx context.Context, task *models.ScheduledTask) error
	Update(ctx context.Context, task *models.ScheduledTask) error
	Delete(ctx context.Context, uid string) error
	GetByUID(ctx context.Context, uid string) (*models.ScheduledTask, error)
	GetByName(ctx context.Context, name string) (*models.ScheduledTask, error)
	List(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*models.ScheduledTask, error)
	Count(ctx context.Context, filters map[string]interface{}) (int64, error)
	ListEnabled(ctx context.Context) ([]*models.ScheduledTask, error)
}

// taskRepository implements TaskRepository interface
type taskRepository struct {
	db *gorm.DB
}

// NewTaskRepository creates a new task repository instance
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{
		db: db,
	}
}

// Create creates a new scheduled task
func (r *taskRepository) Create(ctx context.Context, task *models.ScheduledTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// Update updates an existing scheduled task
func (r *taskRepository) Update(ctx context.Context, task *models.ScheduledTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// Delete deletes a scheduled task by UID
func (r *taskRepository) Delete(ctx context.Context, uid string) error {
	return r.db.WithContext(ctx).
		Where("uid = ?", uid).
		Delete(&models.ScheduledTask{}).Error
}

// GetByUID retrieves a scheduled task by UID
func (r *taskRepository) GetByUID(ctx context.Context, uid string) (*models.ScheduledTask, error) {
	var task models.ScheduledTask
	err := r.db.WithContext(ctx).
		Where("uid = ?", uid).
		First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetByName retrieves a scheduled task by name
func (r *taskRepository) GetByName(ctx context.Context, name string) (*models.ScheduledTask, error) {
	var task models.ScheduledTask
	err := r.db.WithContext(ctx).
		Where("name = ?", name).
		First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// List retrieves a paginated list of scheduled tasks with optional filtering
func (r *taskRepository) List(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*models.ScheduledTask, error) {
	var tasks []*models.ScheduledTask

	query := r.db.WithContext(ctx).Model(&models.ScheduledTask{})

	// Apply filters
	query = r.applyFilters(query, filters)

	// Apply pagination
	err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&tasks).Error

	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// Count counts the total number of scheduled tasks matching the filters
func (r *taskRepository) Count(ctx context.Context, filters map[string]interface{}) (int64, error) {
	var count int64

	query := r.db.WithContext(ctx).Model(&models.ScheduledTask{})

	// Apply filters
	query = r.applyFilters(query, filters)

	err := query.Count(&count).Error
	return count, err
}

// ListEnabled retrieves all enabled scheduled tasks
func (r *taskRepository) ListEnabled(ctx context.Context) ([]*models.ScheduledTask, error) {
	var tasks []*models.ScheduledTask
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("priority DESC, created_at ASC").
		Find(&tasks).Error

	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// applyFilters applies filter conditions to the query
func (r *taskRepository) applyFilters(query *gorm.DB, filters map[string]interface{}) *gorm.DB {
	if enabled, ok := filters["enabled"].(bool); ok {
		query = query.Where("enabled = ?", enabled)
	}

	if taskType, ok := filters["type"].(string); ok && taskType != "" {
		query = query.Where("type = ?", taskType)
	}

	if name, ok := filters["name"].(string); ok && name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	return query
}
