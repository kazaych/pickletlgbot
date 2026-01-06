package models

import (
	"time"

	"gorm.io/gorm"
)

// LocationGORM — таблица `locations` для хранения локаций.
type LocationGORM struct {
	ID            uint   `gorm:"primaryKey" json:"-"`
	LocationID    string `gorm:"uniqueIndex;size:36" json:"-"` // UUID или custom ID
	Name          string `gorm:"size:255;not null" json:"name"`
	Address       string `gorm:"size:500;not null" json:"address"`
	Description   string `gorm:"type:text" json:"description"`
	AddressMapURL string `gorm:"size:500" json:"address_map_url"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}
