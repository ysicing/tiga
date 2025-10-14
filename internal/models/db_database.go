package models

import "github.com/google/uuid"

// Database represents a logical database within a managed database instance.
type Database struct {
	BaseModel

	InstanceID uuid.UUID         `gorm:"type:char(36);not null;index:idx_db_database_instance,priority:1;index:idx_db_database_instance_name,priority:1,unique" json:"instance_id"`
	Instance   *DatabaseInstance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`

	Name       string `gorm:"type:varchar(100);not null;index:idx_db_database_instance_name,priority:2,unique" json:"name"`
	Charset    string `gorm:"type:varchar(50)" json:"charset"`
	Collation  string `gorm:"type:varchar(50)" json:"collation"`
	Owner      string `gorm:"type:varchar(100)" json:"owner"`
	SizeBytes  int64  `gorm:"default:0" json:"size"`
	TableCount int    `gorm:"default:0" json:"table_count"`

	// Redis specific metadata
	DBNumber int `gorm:"default:-1" json:"db_number"`
	KeyCount int `gorm:"default:0" json:"key_count"`

	Permissions []PermissionPolicy `gorm:"foreignKey:DatabaseID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName overrides the default table name.
func (Database) TableName() string {
	return "databases"
}
