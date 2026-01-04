package event

import (
	"context"
	"fmt"
)

const (
	UpdateTypeDec string = "decrement"
	UpdateTypeInc string = "increment"
)

// Service сервис для работы с событиями (бизнес-логика)
// Не знает ни про Redis, ни про Telegram
type Service struct {
	repo Repository
}

// NewService создает новый сервис для работы с событиями
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// CreateEvent создает новое событие
func (s *Service) CreateEvent(ctx context.Context, locationID string, name string) (*Event, error) {
	event := &Event{
		LocationID: locationID,
		Name:       name,
	}

	err := s.repo.CreateEvent(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания события: %w", err)
	}

	return event, nil
}

func (s *Service) DeleteEvent(ctx context.Context, id string) (*Event, error) {
	err := s.repo.DeleteEvent(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ошибка удаления события: %w", err)
	}
	return nil, nil
}

// GetEvent получает событие по ID
func (s *Service) GetEvent(ctx context.Context, eventID string) (*Event, error) {
	event, err := s.repo.GetEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("событие не найдено: %w", err)
	}

	return event, nil
}

// ListEvents получает список событий для локации
func (s *Service) ListEvents(ctx context.Context, locationID string) ([]*Event, error) {
	return s.repo.ListEvents(ctx, locationID)
}

// RegisterUser регистрирует пользователя на событие
func (s *Service) RegisterUser(ctx context.Context, eventID string, userID int64) error {
	return s.repo.RegisterUser(ctx, eventID, userID)
}

// ListUserEvents получает список событий пользователя
func (s *Service) ListUserEvents(ctx context.Context, userID int64) ([]*Event, error) {
	return s.repo.ListUserEvents(ctx, userID)
}

// UpdateEventRemaining обновляет количество оставшихся мест на событии
func (s *Service) UpdateEventRemaining(ctx context.Context, eventID string, updType string) error {
	switch updType {
	case UpdateTypeInc:
		event, err := s.repo.GetEvent(ctx, eventID)
		if err != nil {
			return fmt.Errorf("событие не найдено: %w", err)
		}
		event.Remaining = event.Remaining + 1
	case UpdateTypeDec:
		event, err := s.repo.GetEvent(ctx, eventID)
		if err != nil {
			return fmt.Errorf("событие не найдено: %w", err)
		}
		event.Remaining = event.Remaining - 1
	}
	return nil
}
