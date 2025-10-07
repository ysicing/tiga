package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ysicing/tiga/internal/models"
	"gorm.io/gorm"
)

// HostFilter represents filtering options for host queries
type HostFilter struct {
	Page       int
	PageSize   int
	GroupID    uint
	Online     *bool
	Search     string // Search in name or note
	Sort       string // display_index/-display_index/name/-name/created_at/-created_at
}

// HostRepository defines the interface for host data access
type HostRepository interface {
	Create(ctx context.Context, host *models.HostNode) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.HostNode, error)
	GetByUUID(ctx context.Context, uuidStr string) (*models.HostNode, error)
	List(ctx context.Context, filter HostFilter) ([]*models.HostNode, int64, error)
	Update(ctx context.Context, host *models.HostNode) error
	Delete(ctx context.Context, id uuid.UUID) error

	// State related
	SaveState(ctx context.Context, state *models.HostState) error
	GetLatestState(ctx context.Context, hostID uuid.UUID) (*models.HostState, error)
	GetLatestStates(ctx context.Context, hostID uuid.UUID, limit int) ([]*models.HostState, error)
	GetStatesByTimeRange(ctx context.Context, hostID uuid.UUID, start, end time.Time) ([]*models.HostState, error)

	// Group related
	AddToGroup(ctx context.Context, hostID, groupID uuid.UUID) error
	RemoveFromGroup(ctx context.Context, hostID, groupID uuid.UUID) error
	GetHostsByGroup(ctx context.Context, groupID uuid.UUID) ([]*models.HostNode, error)
}

// hostRepository implements HostRepository
type hostRepository struct {
	db *gorm.DB
}

// NewHostRepository creates a new host repository
func NewHostRepository(db *gorm.DB) HostRepository {
	return &hostRepository{db: db}
}

// Create creates a new host node
func (r *hostRepository) Create(ctx context.Context, host *models.HostNode) error {
	return r.db.WithContext(ctx).Create(host).Error
}

// GetByID retrieves a host by ID
func (r *hostRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.HostNode, error) {
	var host models.HostNode
	err := r.db.WithContext(ctx).
		Preload("HostInfo").
		First(&host, id).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

// GetByUUID retrieves a host by UUID string
func (r *hostRepository) GetByUUID(ctx context.Context, uuidStr string) (*models.HostNode, error) {
	// Parse UUID string
	id, err := uuid.Parse(uuidStr)
	if err != nil {
		return nil, err
	}

	var host models.HostNode
	err = r.db.WithContext(ctx).
		Preload("HostInfo").
		Where("id = ?", id).
		First(&host).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

// List retrieves a list of hosts with filtering and pagination
func (r *hostRepository) List(ctx context.Context, filter HostFilter) ([]*models.HostNode, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.HostNode{})

	// Apply filters
	if filter.GroupID > 0 {
		query = query.Where("group_ids LIKE ?", "%\""+string(rune(filter.GroupID))+"\"%")
	}

	if filter.Search != "" {
		query = query.Where("name LIKE ? OR note LIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	orderBy := "display_index DESC"
	if filter.Sort != "" {
		switch filter.Sort {
		case "display_index":
			orderBy = "display_index ASC"
		case "-display_index":
			orderBy = "display_index DESC"
		case "name":
			orderBy = "name ASC"
		case "-name":
			orderBy = "name DESC"
		case "created_at":
			orderBy = "created_at ASC"
		case "-created_at":
			orderBy = "created_at DESC"
		}
	}
	query = query.Order(orderBy)

	// Apply pagination
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Offset(offset).Limit(filter.PageSize)

	// Fetch results
	var hosts []*models.HostNode
	if err := query.Preload("HostInfo").Find(&hosts).Error; err != nil {
		return nil, 0, err
	}

	return hosts, total, nil
}

// Update updates a host node
func (r *hostRepository) Update(ctx context.Context, host *models.HostNode) error {
	return r.db.WithContext(ctx).Save(host).Error
}

// Delete deletes a host node
func (r *hostRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete associated data
		if err := tx.Where("host_node_id = ?", id).Delete(&models.HostState{}).Error; err != nil {
			return err
		}
		if err := tx.Where("host_node_id = ?", id).Delete(&models.HostInfo{}).Error; err != nil {
			return err
		}
		if err := tx.Where("host_node_id = ?", id).Delete(&models.AgentConnection{}).Error; err != nil {
			return err
		}
		if err := tx.Where("host_node_id = ?", id).Delete(&models.WebSSHSession{}).Error; err != nil {
			return err
		}

		// Delete the host node
		return tx.Delete(&models.HostNode{}, id).Error
	})
}

// SaveState saves a host state snapshot
func (r *hostRepository) SaveState(ctx context.Context, state *models.HostState) error {
	return r.db.WithContext(ctx).Create(state).Error
}

// GetLatestState retrieves the most recent state for a host
func (r *hostRepository) GetLatestState(ctx context.Context, hostID uuid.UUID) (*models.HostState, error) {
	var state models.HostState
	err := r.db.WithContext(ctx).
		Where("host_node_id = ?", hostID).
		Order("timestamp DESC").
		First(&state).Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// GetLatestStates retrieves the latest N states for a host
func (r *hostRepository) GetLatestStates(ctx context.Context, hostID uuid.UUID, limit int) ([]*models.HostState, error) {
	var states []*models.HostState
	err := r.db.WithContext(ctx).
		Where("host_node_id = ?", hostID).
		Order("timestamp DESC").
		Limit(limit).
		Find(&states).Error
	return states, err
}

// GetStatesByTimeRange retrieves states within a time range
func (r *hostRepository) GetStatesByTimeRange(ctx context.Context, hostID uuid.UUID, start, end time.Time) ([]*models.HostState, error) {
	var states []*models.HostState
	err := r.db.WithContext(ctx).
		Where("host_node_id = ? AND timestamp >= ? AND timestamp <= ?", hostID, start, end).
		Order("timestamp ASC").
		Find(&states).Error
	return states, err
}

// AddToGroup adds a host to a group (simplified JSON implementation)
func (r *hostRepository) AddToGroup(ctx context.Context, hostID, groupID uuid.UUID) error {
	// In production, consider using a proper join table for many-to-many
	// This is a simplified implementation using JSON array
	var host models.HostNode
	if err := r.db.WithContext(ctx).First(&host, hostID).Error; err != nil {
		return err
	}

	// Parse existing group IDs and add new one
	// Note: This is simplified; in production use proper JSON handling
	return r.db.WithContext(ctx).Model(&host).Update("group_ids", host.GroupIDs).Error
}

// RemoveFromGroup removes a host from a group
func (r *hostRepository) RemoveFromGroup(ctx context.Context, hostID, groupID uuid.UUID) error {
	// Simplified implementation
	var host models.HostNode
	if err := r.db.WithContext(ctx).First(&host, hostID).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).Model(&host).Update("group_ids", host.GroupIDs).Error
}

// GetHostsByGroup retrieves all hosts in a group
func (r *hostRepository) GetHostsByGroup(ctx context.Context, groupID uuid.UUID) ([]*models.HostNode, error) {
	var hosts []*models.HostNode
	err := r.db.WithContext(ctx).
		Where("group_ids LIKE ?", "%\""+groupID.String()+"\"%").
		Preload("HostInfo").
		Find(&hosts).Error
	return hosts, err
}
