package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// HostNode represents a monitored host server
type HostNode struct {
	BaseModel

	// Basic information
	Name      string `gorm:"not null" json:"name"`
	SecretKey string `gorm:"not null" json:"-"` // Encrypted with AES-256, never expose in JSON

	// Display configuration
	Note         string `gorm:"type:text" json:"note"`
	PublicNote   string `gorm:"type:text" json:"public_note"`
	DisplayIndex int    `gorm:"default:0;index" json:"display_index"`
	HideForGuest bool   `gorm:"default:false" json:"hide_for_guest"`

	// Billing and expiry information
	Cost         float64    `gorm:"default:0" json:"cost"`               // 费用（根据续费周期决定是月费还是年费）
	RenewalType  string     `gorm:"default:monthly" json:"renewal_type"` // 续费周期：monthly 或 yearly
	PurchaseDate *time.Time `gorm:"index" json:"purchase_date,omitempty"` // 购买日期
	ExpiryDate   *time.Time `gorm:"index" json:"expiry_date,omitempty"`  // 到期时间
	AutoRenew    bool       `gorm:"default:false" json:"auto_renew"`     // 是否自动续费
	TrafficLimit int64      `gorm:"default:0" json:"traffic_limit"`      // 流量限制 (GB), 0表示无限
	TrafficUsed  int64      `gorm:"default:0" json:"traffic_used"`       // 已用流量 (GB)

	// Group associations
	GroupID  *uuid.UUID `gorm:"type:char(36);index" json:"group_id,omitempty"` // 主分组ID
	GroupIDs string     `gorm:"type:text" json:"group_ids"`                    // 多分组支持（JSON array）

	// Runtime status (not persisted to database)
	Online     bool       `gorm:"-" json:"online"`
	LastActive *time.Time `gorm:"-" json:"last_active,omitempty"`

	// Relationships
	HostInfo        *HostInfo          `gorm:"foreignKey:HostNodeID" json:"host_info,omitempty"`
	States          []HostState        `gorm:"foreignKey:HostNodeID" json:"-"`
	WebSSHSessions  []WebSSHSession    `gorm:"foreignKey:HostNodeID" json:"-"`
	ServiceMonitors []ServiceMonitor   `gorm:"foreignKey:HostNodeID" json:"-"`
	AlertRules      []MonitorAlertRule `gorm:"foreignKey:TargetID" json:"-"`
}

// TableName specifies the table name for HostNode
func (HostNode) TableName() string {
	return "host_nodes"
}

// BeforeCreate validates the model before creation
func (h *HostNode) BeforeCreate(tx *gorm.DB) error {
	// Set default renewal type if not specified
	if h.RenewalType == "" {
		h.RenewalType = "monthly"
	}
	return nil
}
