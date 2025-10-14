package models

import "time"

// DatabaseInstance represents an external database server that Tiga manages.
type DatabaseInstance struct {
	BaseModel

	Name        string     `gorm:"type:varchar(100);not null;uniqueIndex:idx_db_instance_name" json:"name"`
	Type        string     `gorm:"type:varchar(20);not null;index:idx_db_instance_type" json:"type"`
	Host        string     `gorm:"type:varchar(255);not null" json:"host"`
	Port        int        `gorm:"not null" json:"port"`
	Username    string     `gorm:"type:varchar(100)" json:"username"`
	Password    string     `gorm:"type:text" json:"-"` // encrypted credential blob
	SSLMode     string     `gorm:"type:varchar(20);default:disable" json:"ssl_mode"`
	Description string     `gorm:"type:varchar(500)" json:"description"`
	Status      string     `gorm:"type:varchar(20);default:pending;index:idx_db_instance_status" json:"status"`
	LastCheckAt *time.Time `json:"last_check_at,omitempty"`
	Version     string     `gorm:"type:varchar(50)" json:"version"`
	Uptime      int64      `gorm:"default:0" json:"uptime"`

	Databases []Database     `gorm:"foreignKey:InstanceID;constraint:OnDelete:CASCADE" json:"-"`
	Users     []DatabaseUser `gorm:"foreignKey:InstanceID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName overrides the default table name.
func (DatabaseInstance) TableName() string {
	return "db_instances"
}
