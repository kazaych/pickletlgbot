package models

import (
	"time"

	"gorm.io/gorm"
)

// EventGORM — таблица `events` для хранения событий
type EventGORM struct {
	ID          uint      `gorm:"primaryKey" json:"-"`
	EventID     string    `gorm:"uniqueIndex;size:36" json:"-"` // UUID
	Name        string    `gorm:"size:255;not null" json:"name"`
	Type        string    `gorm:"size:50;not null" json:"type"` // training, competition
	Date        time.Time `gorm:"not null" json:"date"`
	Remaining   int       `gorm:"not null;default:0" json:"remaining"`
	MaxPlayers  int       `gorm:"not null" json:"max_players"`
	Players     string    `gorm:"type:text" json:"players"` // JSON массив int64 (только approved)
	LocationID  string    `gorm:"size:36;not null;index" json:"location_id"`
	Trainer     string    `gorm:"size:255" json:"trainer"` // Тренер события
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// EventRegistrationGORM — таблица для хранения регистраций пользователей на события
type EventRegistrationGORM struct {
	ID        uint   `gorm:"primaryKey" json:"-"`
	EventID   string `gorm:"size:36;not null;uniqueIndex:idx_event_user" json:"event_id"`
	UserID    int64  `gorm:"not null;uniqueIndex:idx_event_user" json:"user_id"`
	Status    string `gorm:"size:20;not null;default:'pending'" json:"status"` // pending, approved, rejected
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Уникальный индекс на пару (EventID, UserID) - один пользователь может быть зарегистрирован на событие только один раз
}
