package models

import (
	"gorm.io/gorm"
)

// HostGroup represents a logical grouping of hosts
type HostGroup struct {
	gorm.Model

	Name        string `gorm:"uniqueIndex;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`

	// Relationships - Many-to-Many with HostNode
	// Implemented through GroupIDs JSON field in HostNode for simplicity
	// This model is used for group metadata and queries
}

// TableName specifies the table name for HostGroup
func (HostGroup) TableName() string {
	return "host_groups"
}

// GetHostCount returns the number of hosts in this group
func (g *HostGroup) GetHostCount(db *gorm.DB) (int64, error) {
	var count int64
	// This requires a JSON query, simplified here
	// In production, consider using a join table for many-to-many
	err := db.Model(&HostNode{}).
		Where("group_ids LIKE ?", "%\""+string(rune(g.ID))+"\"%").
		Count(&count).Error
	return count, err
}
