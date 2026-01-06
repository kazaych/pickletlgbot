package event

import (
	"context"

	"pickletlgbot/internal/domain/location"
)

// EventRepository описывает, что нужно домену от хранилища событий
type EventRepository interface {
	// GetByID возвращает событие по ID или ошибку, если не найдено
	GetByID(ctx context.Context, id EventID) (*Event, error)

	// List возвращает все доступные события
	List(ctx context.Context) ([]Event, error)

	// ListByLocation возвращает события для конкретной локации
	ListByLocation(ctx context.Context, locationID location.LocationID) ([]Event, error)

	// ListByUser возвращает события, на которые зарегистрирован пользователь
	ListByUser(ctx context.Context, userID int64) ([]Event, error)

	// Save создаёт или обновляет событие
	Save(ctx context.Context, event *Event) error

	// Delete удаляет событие по ID
	Delete(ctx context.Context, id EventID) error
}
