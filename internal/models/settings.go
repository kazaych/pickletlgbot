package models

import "gorm.io/gorm"

type SettingsGORM struct {
	gorm.Model
	Key   string `gorm:"uniqueIndex;not null"`
	Value string  `gorm:"not null"`
}
