package event

import (
	"errors"
	"time"

	"pickletlgbot/internal/domain/location"
)

// EventID - тип для ID события (аналогично LocationID)
type EventID string

// RegistrationStatus - статус регистрации пользователя
type RegistrationStatus string

const (
	RegistrationStatusPending  RegistrationStatus = "pending"  // Ожидает подтверждения
	RegistrationStatusApproved RegistrationStatus = "approved" // Подтвержден
	RegistrationStatusRejected RegistrationStatus = "rejected" // Отклонен
)

// EventRegistration - регистрация пользователя на событие
type EventRegistration struct {
	UserID    int64
	Status    RegistrationStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Event представляет событие в доменной модели
type Event struct {
	ID            EventID
	Name          string
	Type          EventType
	Date          time.Time
	Remaining     int                         // Количество оставшихся мест
	MaxPlayers    int                         // Максимальное количество игроков
	Players       []int64                     // ID подтвержденных пользователей Telegram
	Registrations map[int64]EventRegistration // Все регистрации (pending + approved + rejected)
	LocationID    location.LocationID
	Trainer       string // Тренер события
	Description   string // Описание события (опционально)
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type EventType string

const (
	EventTypeTraining    EventType = "training"
	EventTypeCompetition EventType = "competition"
)

// CreateEventInput - DTO для создания события
type CreateEventInput struct {
	Name        string
	Type        EventType
	Date        time.Time
	MaxPlayers  int
	LocationID  location.LocationID
	Trainer     string
	Description string
}

// UpdateEventInput - DTO для обновления события
type UpdateEventInput struct {
	Name        *string
	Type        *EventType
	Date        *time.Time
	MaxPlayers  *int
	Remaining   *int
	Description *string
}

// Validate проверяет валидность входных данных для создания события
func (in CreateEventInput) Validate() error {
	if in.Name == "" {
		return ErrEventNameRequired
	}
	if in.LocationID == "" {
		return ErrLocationIDRequired
	}
	if in.Date.IsZero() {
		return ErrDateRequired
	}
	if in.Date.Before(time.Now()) {
		return ErrDateInPast
	}
	if in.MaxPlayers <= 0 {
		return ErrMaxPlayersInvalid
	}
	return nil
}

// Errors
var (
	ErrEventNameRequired           = errors.New("event name is required")
	ErrLocationIDRequired          = errors.New("location ID is required")
	ErrDateRequired                = errors.New("event date is required")
	ErrDateInPast                  = errors.New("event date cannot be in the past")
	ErrMaxPlayersInvalid           = errors.New("max players must be greater than 0")
	ErrEventNotFound               = errors.New("event not found")
	ErrEventFull                   = errors.New("event is full")
	ErrUserAlreadyRegistered       = errors.New("user is already registered for this event")
	ErrRegistrationNotFound        = errors.New("registration not found")
	ErrRegistrationAlreadyApproved = errors.New("registration already approved")
	ErrRegistrationAlreadyRejected = errors.New("registration already rejected")
)
