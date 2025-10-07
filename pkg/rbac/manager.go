package rbac

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/pkg/common"
)

var (
	RBACConfig *common.RolesConfig
	once       sync.Once
	rwlock     sync.RWMutex
	globalDB   *gorm.DB
)

// InitRBAC initializes the RBAC system with database
// Simplified: No longer loads roles from database, using is_admin flag instead
func InitRBAC(db *gorm.DB) {
	globalDB = db
	once.Do(func() {
		// Initialize empty config as RBAC system is simplified
		RBACConfig = &common.RolesConfig{
			Roles:       []common.Role{},
			RoleMapping: []common.RoleMapping{},
		}
		logrus.Info("RBAC system initialized in simplified mode (using is_admin flag)")
		// Start background sync goroutine (does nothing now but prevents panics if code sends to SyncNow)
		go SyncRolesConfig()
	})
}

// InitDefaultRole creates default roles in database
// Simplified: No-op as role system is removed
func InitDefaultRole(db *gorm.DB) error {
	logrus.Debug("InitDefaultRole skipped - using simplified RBAC based on is_admin flag")
	return nil
}

// loadRolesFromDB populates RBACConfig from DB rows
// Simplified: No-op as role system is removed
func loadRolesFromDB() error {
	// Keep empty config
	rwlock.Lock()
	if RBACConfig == nil {
		RBACConfig = &common.RolesConfig{
			Roles:       []common.Role{},
			RoleMapping: []common.RoleMapping{},
		}
	}
	rwlock.Unlock()
	return nil
}

var (
	SyncNow = make(chan struct{}, 1)
)

func SyncRolesConfig() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	// Drain initial sync signal
	<-SyncNow
	for {
		select {
		case <-ticker.C:
			// No-op: simplified RBAC doesn't need sync
		case <-SyncNow:
			// No-op: simplified RBAC doesn't need sync
			// But we drain the channel to prevent blocking senders
		}
	}
}
