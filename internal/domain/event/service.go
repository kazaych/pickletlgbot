package event

import (
	"context"
	"errors"
	"time"

	"pickletlgbot/internal/domain/location"

	"github.com/google/uuid"
)

// EventService описывает use-case'ы вокруг событий (аналогично LocationService)
type EventService interface {
	Get(ctx context.Context, id EventID) (*Event, error)
	List(ctx context.Context) ([]Event, error)
	ListByLocation(ctx context.Context, locationID location.LocationID) ([]Event, error)
	ListByUser(ctx context.Context, userID int64) ([]Event, error)
	Create(ctx context.Context, input CreateEventInput) (*Event, error)
	Update(ctx context.Context, id EventID, input UpdateEventInput) (*Event, error)
	Delete(ctx context.Context, id EventID) error

	// Регистрация пользователей
	RegisterUserToEvent(ctx context.Context, eventID EventID, userID int64) error // Создает регистрацию со статусом pending
	UnregisterUser(ctx context.Context, eventID EventID, userID int64) error

	// Модерация регистраций (для админов)
	ApproveRegistration(ctx context.Context, eventID EventID, userID int64) error
	RejectRegistration(ctx context.Context, eventID EventID, userID int64) error
	ListPendingRegistrations(ctx context.Context, eventID EventID) ([]EventRegistration, error)
}

type eventService struct {
	repo            EventRepository
	locationService location.LocationService // Для валидации локации
}

func NewEventService(repo EventRepository, locationService location.LocationService) EventService {
	return &eventService{
		repo:            repo,
		locationService: locationService,
	}
}

func (s *eventService) Get(ctx context.Context, id EventID) (*Event, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *eventService) List(ctx context.Context) ([]Event, error) {
	return s.repo.List(ctx)
}

func (s *eventService) ListByLocation(ctx context.Context, locationID location.LocationID) ([]Event, error) {
	return s.repo.ListByLocation(ctx, locationID)
}

func (s *eventService) ListByUser(ctx context.Context, userID int64) ([]Event, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *eventService) Create(ctx context.Context, in CreateEventInput) (*Event, error) {
	// Валидация входных данных
	if err := in.Validate(); err != nil {
		return nil, err
	}

	// Проверяем, что локация существует
	_, err := s.locationService.Get(ctx, in.LocationID)
	if err != nil {
		return nil, errors.New("location not found")
	}

	// Создаем событие
	event := &Event{
		ID:            EventID(uuid.New().String()),
		Name:          in.Name,
		Type:          in.Type,
		Date:          in.Date,
		MaxPlayers:    in.MaxPlayers,
		Remaining:     in.MaxPlayers, // Изначально все места свободны
		Players:       []int64{},
		Registrations: make(map[int64]EventRegistration),
		LocationID:    in.LocationID,
		Trainer:       in.Trainer,
		Description:   in.Description,
		PaymentPhone:  in.PaymentPhone,
		Price:         in.Price,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Save(ctx, event); err != nil {
		return nil, err
	}

	return event, nil
}

func (s *eventService) Update(ctx context.Context, id EventID, in UpdateEventInput) (*Event, error) {
	event, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, ErrEventNotFound
	}

	// Обновляем поля
	if in.Name != nil {
		event.Name = *in.Name
	}
	if in.Type != nil {
		event.Type = *in.Type
	}
	if in.Date != nil {
		event.Date = *in.Date
	}
	if in.MaxPlayers != nil {
		// При изменении MaxPlayers нужно пересчитать Remaining
		oldMax := event.MaxPlayers
		event.MaxPlayers = *in.MaxPlayers
		// Если увеличили количество мест, увеличиваем и оставшиеся
		if *in.MaxPlayers > oldMax {
			event.Remaining += (*in.MaxPlayers - oldMax)
		} else if *in.MaxPlayers < oldMax {
			// Если уменьшили, уменьшаем оставшиеся (но не меньше 0)
			diff := oldMax - *in.MaxPlayers
			if event.Remaining > diff {
				event.Remaining -= diff
			} else {
				event.Remaining = 0
			}
		}
	}
	if in.Remaining != nil {
		event.Remaining = *in.Remaining
	}
	if in.Description != nil {
		event.Description = *in.Description
	}

	event.UpdatedAt = time.Now()

	if err := s.repo.Save(ctx, event); err != nil {
		return nil, err
	}

	return event, nil
}

func (s *eventService) Delete(ctx context.Context, id EventID) error {
	return s.repo.Delete(ctx, id)
}

func (s *eventService) RegisterUserToEvent(ctx context.Context, eventID EventID, userID int64) error {
	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		return err
	}
	if event == nil {
		return ErrEventNotFound
	}

	// Проверяем, не зарегистрирован ли уже пользователь (в любом статусе)
	if reg, exists := event.Registrations[userID]; exists {
		if reg.Status == RegistrationStatusPending {
			return ErrUserAlreadyRegistered // Уже есть pending регистрация
		}
		if reg.Status == RegistrationStatusApproved {
			return ErrUserAlreadyRegistered // Уже подтвержден
		}
		// Если был rejected, можно зарегистрироваться снова
	}

	// Создаем регистрацию со статусом pending
	event.Registrations[userID] = EventRegistration{
		UserID:    userID,
		Status:    RegistrationStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	event.UpdatedAt = time.Now()

	return s.repo.Save(ctx, event)
}

func (s *eventService) UnregisterUser(ctx context.Context, eventID EventID, userID int64) error {
	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		return err
	}
	if event == nil {
		return ErrEventNotFound
	}

	// Проверяем, существует ли регистрация
	reg, exists := event.Registrations[userID]
	if !exists {
		return ErrRegistrationNotFound
	}

	// Если был approved, убираем из списка игроков
	if reg.Status == RegistrationStatusApproved {
		for i, playerID := range event.Players {
			if playerID == userID {
				event.Players = append(event.Players[:i], event.Players[i+1:]...)
				event.Remaining++
				break
			}
		}
	}

	// Удаляем регистрацию
	delete(event.Registrations, userID)
	event.UpdatedAt = time.Now()

	return s.repo.Save(ctx, event)
}

func (s *eventService) ApproveRegistration(ctx context.Context, eventID EventID, userID int64) error {
	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		return err
	}
	if event == nil {
		return ErrEventNotFound
	}

	// Проверяем, существует ли регистрация
	reg, exists := event.Registrations[userID]
	if !exists {
		return ErrRegistrationNotFound
	}

	// Проверяем статус
	if reg.Status == RegistrationStatusApproved {
		return ErrRegistrationAlreadyApproved
	}
	if reg.Status == RegistrationStatusRejected {
		return errors.New("cannot approve rejected registration")
	}

	// Проверяем, есть ли свободные места
	if event.Remaining <= 0 {
		return ErrEventFull
	}

	// Обновляем статус регистрации
	reg.Status = RegistrationStatusApproved
	reg.UpdatedAt = time.Now()
	event.Registrations[userID] = reg

	// Добавляем пользователя в список подтвержденных игроков
	event.Players = append(event.Players, userID)
	event.Remaining--
	event.UpdatedAt = time.Now()

	return s.repo.Save(ctx, event)
}

func (s *eventService) RejectRegistration(ctx context.Context, eventID EventID, userID int64) error {
	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		return err
	}
	if event == nil {
		return ErrEventNotFound
	}

	// Проверяем, существует ли регистрация
	reg, exists := event.Registrations[userID]
	if !exists {
		return ErrRegistrationNotFound
	}

	// Проверяем статус
	if reg.Status == RegistrationStatusRejected {
		return ErrRegistrationAlreadyRejected
	}
	if reg.Status == RegistrationStatusApproved {
		// Если был подтвержден, нужно убрать из списка игроков
		for i, playerID := range event.Players {
			if playerID == userID {
				event.Players = append(event.Players[:i], event.Players[i+1:]...)
				event.Remaining++
				break
			}
		}
	}

	// Обновляем статус регистрации
	reg.Status = RegistrationStatusRejected
	reg.UpdatedAt = time.Now()
	event.Registrations[userID] = reg
	event.UpdatedAt = time.Now()

	return s.repo.Save(ctx, event)
}

func (s *eventService) ListPendingRegistrations(ctx context.Context, eventID EventID) ([]EventRegistration, error) {
	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, ErrEventNotFound
	}

	var pending []EventRegistration
	for _, reg := range event.Registrations {
		if reg.Status == RegistrationStatusPending {
			pending = append(pending, reg)
		}
	}

	return pending, nil
}
