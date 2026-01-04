package event

import "context"

// Repository интерфейс для работы с событиями в хранилище
type Repository interface {
	// CreateEvent создает новое событие
	CreateEvent(ctx context.Context, event *Event) error

	// DeleteEvent удаляет событие
	DeleteEvent(ctx context.Context, eventID string) error

	// GetEvent получает событие по ID
	GetEvent(ctx context.Context, eventID string) (*Event, error)

	// ListEvents получает список событий
	ListEvents(ctx context.Context, locationID string) ([]*Event, error)

	// RegisterUser регистрирует пользователя на событие
	RegisterUser(ctx context.Context, eventID string, userID int64) error

	// ListUserEvents получает список событий пользователя
	ListUserEvents(ctx context.Context, userID int64) ([]*Event, error)

	// UpdateEventRemaining обновляет количество оставшихся мест на событии
	UpdateEventRemaining(ctx context.Context, eventID string, updType string) error
}
