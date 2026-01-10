package models

import (
	"time"

	"gorm.io/gorm"
)

// UserGORM — таблица `user` для хранения пользователей
type UserGORM struct {
	ID         int64          `gorm:"primaryKey;autoIncrement" json:"id"` // Автоинкрементный первичный ключ (BIGSERIAL)
	Name       string         `gorm:"size:255;not null" json:"name"`
	Surname    string         `gorm:"size:255" json:"surname"`
	TelegramID int64          `gorm:"uniqueIndex;not null" json:"telegram_id"` // Уникальный идентификатор Telegram
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}
