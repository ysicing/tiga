package models

import (
	"gorm.io/gorm"
)

// Global DB for backward compatibility with legacy code
var DB *gorm.DB

// InitRepositories initializes the global DB variable
// Note: Individual repositories should be created where needed to avoid import cycles
func InitRepositories(db *gorm.DB) {
	DB = db
}
