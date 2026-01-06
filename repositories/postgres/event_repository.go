package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"pickletlgbot/internal/domain/event"
	"pickletlgbot/internal/domain/location"
	"pickletlgbot/internal/models"

	"gorm.io/gorm"
)

type eventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) event.EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) GetByID(ctx context.Context, id event.EventID) (*event.Event, error) {
	var model models.EventGORM
	if err := r.db.WithContext(ctx).
		Where("event_id = ?", string(id)).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	evt, err := r.modelToDomain(&model)
	if err != nil {
		return nil, err
	}

	// Загружаем регистрации из отдельной таблицы
	registrations, err := r.loadRegistrations(ctx, id)
	if err != nil {
		return nil, err
	}
	evt.Registrations = registrations

	// Пересчитываем Players и Remaining на основе approved регистраций
	r.recalculatePlayersAndRemaining(evt)

	return evt, nil
}

func (r *eventRepository) List(ctx context.Context) ([]event.Event, error) {
	var models []models.EventGORM
	if err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Find(&models).Error; err != nil {
		return nil, err
	}

	events := make([]event.Event, len(models))
	for i, m := range models {
		evt, err := r.modelToDomain(&m)
		if err != nil {
			return nil, err
		}
		// Загружаем регистрации для каждого события
		registrations, err := r.loadRegistrations(ctx, event.EventID(m.EventID))
		if err != nil {
			return nil, err
		}
		evt.Registrations = registrations
		// Пересчитываем Players и Remaining на основе approved регистраций
		r.recalculatePlayersAndRemaining(evt)
		events[i] = *evt
	}
	return events, nil
}

func (r *eventRepository) ListByLocation(ctx context.Context, locationID location.LocationID) ([]event.Event, error) {
	var models []models.EventGORM
	if err := r.db.WithContext(ctx).
		Where("location_id = ? AND deleted_at IS NULL", string(locationID)).
		Find(&models).Error; err != nil {
		return nil, err
	}

	events := make([]event.Event, len(models))
	for i, m := range models {
		evt, err := r.modelToDomain(&m)
		if err != nil {
			return nil, err
		}
		// Загружаем регистрации для каждого события
		registrations, err := r.loadRegistrations(ctx, event.EventID(m.EventID))
		if err != nil {
			return nil, err
		}
		evt.Registrations = registrations
		// Пересчитываем Players и Remaining на основе approved регистраций
		r.recalculatePlayersAndRemaining(evt)
		events[i] = *evt
	}
	return events, nil
}

func (r *eventRepository) ListByUser(ctx context.Context, userID int64) ([]event.Event, error) {
	// Ищем события через таблицу регистраций
	var registrations []models.EventRegistrationGORM
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Find(&registrations).Error; err != nil {
		return nil, err
	}

	// Собираем уникальные event_id
	eventIDs := make(map[string]bool)
	for _, reg := range registrations {
		eventIDs[reg.EventID] = true
	}

	if len(eventIDs) == 0 {
		return []event.Event{}, nil
	}

	// Загружаем события
	var eventIDList []string
	for id := range eventIDs {
		eventIDList = append(eventIDList, id)
	}

	var models []models.EventGORM
	if err := r.db.WithContext(ctx).
		Where("event_id IN ? AND deleted_at IS NULL", eventIDList).
		Find(&models).Error; err != nil {
		return nil, err
	}

	userEvents := make([]event.Event, 0, len(models))
	for _, m := range models {
		evt, err := r.modelToDomain(&m)
		if err != nil {
			continue
		}

		// Загружаем регистрации для этого события
		registrations, err := r.loadRegistrations(ctx, event.EventID(m.EventID))
		if err != nil {
			continue
		}
		evt.Registrations = registrations
		// Пересчитываем Players и Remaining на основе approved регистраций
		r.recalculatePlayersAndRemaining(evt)

		userEvents = append(userEvents, *evt)
	}
	return userEvents, nil
}

func (r *eventRepository) Save(ctx context.Context, evt *event.Event) error {
	model, err := r.domainToModel(evt)
	if err != nil {
		return err
	}

	// Используем FirstOrCreate с Assign для создания или обновления (как в location_repository)
	err = r.db.WithContext(ctx).
		Where("event_id = ?", string(evt.ID)).
		Assign(model).
		FirstOrCreate(model).Error

	if err != nil {
		return err
	}

	// Сохраняем регистрации в отдельной таблице
	return r.saveRegistrations(ctx, evt.ID, evt.Registrations)
}

func (r *eventRepository) Delete(ctx context.Context, id event.EventID) error {
	// Удаляем регистрации
	if err := r.db.WithContext(ctx).
		Where("event_id = ?", string(id)).
		Delete(&models.EventRegistrationGORM{}).Error; err != nil {
		return err
	}

	// Удаляем событие
	return r.db.WithContext(ctx).
		Where("event_id = ?", string(id)).
		Delete(&models.EventGORM{}).Error
}

// modelToDomain конвертирует GORM модель в доменную модель
func (r *eventRepository) modelToDomain(model *models.EventGORM) (*event.Event, error) {
	evt := &event.Event{
		ID:          event.EventID(model.EventID),
		Name:        model.Name,
		Type:        event.EventType(model.Type),
		Date:        model.Date,
		Remaining:   model.Remaining,
		MaxPlayers:  model.MaxPlayers,
		LocationID:  location.LocationID(model.LocationID),
		Trainer:     model.Trainer,
		Description: model.Description,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}

	// Парсим Players из JSON
	if model.Players != "" {
		var players []int64
		if err := json.Unmarshal([]byte(model.Players), &players); err != nil {
			// Если ошибка, оставляем пустым
			evt.Players = []int64{}
		} else {
			evt.Players = players
		}
	} else {
		evt.Players = []int64{}
	}

	// Registrations загружаются отдельно через loadRegistrations
	evt.Registrations = make(map[int64]event.EventRegistration)

	return evt, nil
}

// domainToModel конвертирует доменную модель в GORM модель
func (r *eventRepository) domainToModel(evt *event.Event) (*models.EventGORM, error) {
	model := &models.EventGORM{
		EventID:     string(evt.ID),
		Name:        evt.Name,
		Type:        string(evt.Type),
		Date:        evt.Date,
		Remaining:   evt.Remaining,
		MaxPlayers:  evt.MaxPlayers,
		LocationID:  string(evt.LocationID),
		Trainer:     evt.Trainer,
		Description: evt.Description,
		CreatedAt:   evt.CreatedAt,
		UpdatedAt:   evt.UpdatedAt,
	}

	// Сериализуем Players в JSON
	if len(evt.Players) > 0 {
		playersJSON, err := json.Marshal(evt.Players)
		if err != nil {
			return nil, err
		}
		model.Players = string(playersJSON)
	} else {
		model.Players = "[]"
	}

	// Registrations сохраняются отдельно через saveRegistrations
	return model, nil
}

// loadRegistrations загружает регистрации из отдельной таблицы
func (r *eventRepository) loadRegistrations(ctx context.Context, eventID event.EventID) (map[int64]event.EventRegistration, error) {
	var regModels []models.EventRegistrationGORM
	if err := r.db.WithContext(ctx).
		Where("event_id = ? AND deleted_at IS NULL", string(eventID)).
		Find(&regModels).Error; err != nil {
		return nil, err
	}

	registrations := make(map[int64]event.EventRegistration)
	for _, regModel := range regModels {
		registrations[regModel.UserID] = event.EventRegistration{
			UserID:    regModel.UserID,
			Status:    event.RegistrationStatus(regModel.Status),
			CreatedAt: regModel.CreatedAt,
			UpdatedAt: regModel.UpdatedAt,
		}
	}

	return registrations, nil
}

// recalculatePlayersAndRemaining пересчитывает Players и Remaining на основе approved регистраций
func (r *eventRepository) recalculatePlayersAndRemaining(evt *event.Event) {
	// Собираем список approved пользователей
	approvedPlayers := make([]int64, 0)
	for userID, reg := range evt.Registrations {
		if reg.Status == event.RegistrationStatusApproved {
			approvedPlayers = append(approvedPlayers, userID)
		}
	}

	// Обновляем Players и Remaining
	evt.Players = approvedPlayers
	evt.Remaining = evt.MaxPlayers - len(approvedPlayers)
	if evt.Remaining < 0 {
		evt.Remaining = 0
	}
}

// saveRegistrations сохраняет регистрации в отдельную таблицу
func (r *eventRepository) saveRegistrations(ctx context.Context, eventID event.EventID, registrations map[int64]event.EventRegistration) error {
	// Получаем текущие регистрации из БД
	var existingRegs []models.EventRegistrationGORM
	if err := r.db.WithContext(ctx).
		Where("event_id = ?", string(eventID)).
		Find(&existingRegs).Error; err != nil {
		return err
	}

	// Создаем карту существующих регистраций
	existingMap := make(map[int64]bool)
	for _, reg := range existingRegs {
		existingMap[reg.UserID] = true
	}

	// Сохраняем или обновляем регистрации
	for userID, reg := range registrations {
		regModel := &models.EventRegistrationGORM{
			EventID:   string(eventID),
			UserID:    userID,
			Status:    string(reg.Status),
			CreatedAt: reg.CreatedAt,
			UpdatedAt: reg.UpdatedAt,
		}

		if existingMap[userID] {
			// Обновляем существующую
			if err := r.db.WithContext(ctx).
				Model(&models.EventRegistrationGORM{}).
				Where("event_id = ? AND user_id = ?", string(eventID), userID).
				Updates(regModel).Error; err != nil {
				return err
			}
		} else {
			// Создаем новую
			if err := r.db.WithContext(ctx).Create(regModel).Error; err != nil {
				return err
			}
		}
	}

	// Удаляем регистрации, которых нет в новой карте
	var userIDsToKeep []int64
	for userID := range registrations {
		userIDsToKeep = append(userIDsToKeep, userID)
	}

	if len(userIDsToKeep) > 0 {
		if err := r.db.WithContext(ctx).
			Where("event_id = ? AND user_id NOT IN ?", string(eventID), userIDsToKeep).
			Delete(&models.EventRegistrationGORM{}).Error; err != nil {
			return err
		}
	} else {
		// Если нет регистраций, удаляем все для этого события
		if err := r.db.WithContext(ctx).
			Where("event_id = ?", string(eventID)).
			Delete(&models.EventRegistrationGORM{}).Error; err != nil {
			return err
		}
	}

	return nil
}
