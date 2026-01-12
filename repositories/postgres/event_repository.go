package postgres

import (
	"context"
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
		registrations, err := r.loadRegistrations(ctx, event.EventID(m.EventID))
		if err != nil {
			return nil, err
		}
		evt.Registrations = registrations
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

	events := make([]event.Event, 0, len(models))
	for _, m := range models {
		evt, err := r.modelToDomain(&m)
		if err != nil {
			return nil, err
		}
		registrations, err := r.loadRegistrations(ctx, event.EventID(m.EventID))
		if err != nil {
			return nil, err
		}
		evt.Registrations = registrations
		r.recalculatePlayersAndRemaining(evt)
		events = append(events, *evt)
	}
	return events, nil
}

func (r *eventRepository) ListByUser(ctx context.Context, userID int64) ([]event.Event, error) {
	var registrations []models.EventRegistrationGORM
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Find(&registrations).Error; err != nil {
		return nil, err
	}

	eventIDs := make(map[string]bool)
	for _, reg := range registrations {
		eventIDs[reg.EventID] = true
	}

	if len(eventIDs) == 0 {
		return []event.Event{}, nil
	}

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
		registrations, err := r.loadRegistrations(ctx, event.EventID(m.EventID))
		if err != nil {
			continue
		}
		evt.Registrations = registrations
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

	err = r.db.WithContext(ctx).
		Where("event_id = ?", string(evt.ID)).
		Assign(model).
		FirstOrCreate(model).Error

	if err != nil {
		return err
	}

	return r.saveRegistrations(ctx, evt.ID, evt.Registrations)
}

func (r *eventRepository) Delete(ctx context.Context, id event.EventID) error {
	if err := r.db.WithContext(ctx).
		Where("event_id = ?", string(id)).
		Delete(&models.EventRegistrationGORM{}).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Where("event_id = ?", string(id)).
		Delete(&models.EventGORM{}).Error
}

func (r *eventRepository) modelToDomain(model *models.EventGORM) (*event.Event, error) {
	evt := &event.Event{
		ID:           event.EventID(model.EventID),
		Name:         model.Name,
		Type:         event.EventType(model.Type),
		Date:         model.Date,
		Remaining:    model.Remaining,
		MaxPlayers:   model.MaxPlayers,
		LocationID:   location.LocationID(model.LocationID),
		Trainer:      model.Trainer,
		Description:  model.Description,
		PaymentPhone: model.PaymentPhone,
		Price:        model.Price,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}

	evt.Players = []int64{}
	evt.Registrations = make(map[int64]event.EventRegistration)

	return evt, nil
}

func (r *eventRepository) domainToModel(evt *event.Event) (*models.EventGORM, error) {
	model := &models.EventGORM{
		EventID:      string(evt.ID),
		Name:         evt.Name,
		Type:         string(evt.Type),
		Date:         evt.Date,
		Remaining:    evt.Remaining,
		MaxPlayers:   evt.MaxPlayers,
		LocationID:   string(evt.LocationID),
		Trainer:      evt.Trainer,
		Description:  evt.Description,
		PaymentPhone: evt.PaymentPhone,
		Price:        evt.Price,
		CreatedAt:    evt.CreatedAt,
		UpdatedAt:    evt.UpdatedAt,
	}

	return model, nil
}

func (r *eventRepository) loadRegistrations(ctx context.Context, eventID event.EventID) (map[int64]event.EventRegistration, error) {
	var regModels []models.EventRegistrationGORM
	if err := r.db.WithContext(ctx).
		Preload("User").
		Where("event_id = ? AND deleted_at IS NULL", string(eventID)).
		Find(&regModels).Error; err != nil {
		return nil, err
	}

	registrations := make(map[int64]event.EventRegistration)
	for _, regModel := range regModels {
		telegramID := regModel.User.TelegramID
		registrations[telegramID] = event.EventRegistration{
			UserID:    telegramID,
			Status:    event.RegistrationStatus(regModel.Status),
			CreatedAt: regModel.CreatedAt,
			UpdatedAt: regModel.UpdatedAt,
		}
	}

	return registrations, nil
}

func (r *eventRepository) recalculatePlayersAndRemaining(evt *event.Event) {
	approvedPlayers := make([]int64, 0)
	for userID, reg := range evt.Registrations {
		if reg.Status == event.RegistrationStatusApproved {
			approvedPlayers = append(approvedPlayers, userID)
		}
	}

	evt.Players = approvedPlayers
	evt.Remaining = evt.MaxPlayers - len(approvedPlayers)
	if evt.Remaining < 0 {
		evt.Remaining = 0
	}
}

func (r *eventRepository) saveRegistrations(ctx context.Context, eventID event.EventID, registrations map[int64]event.EventRegistration) error {
	// Используем Unscoped() чтобы найти и soft-deleted записи
	var existingRegs []models.EventRegistrationGORM
	if err := r.db.WithContext(ctx).Unscoped().
		Preload("User").
		Where("event_id = ?", string(eventID)).
		Find(&existingRegs).Error; err != nil {
		return err
	}

	existingMap := make(map[int64]int64) // telegramID -> userID
	deletedMap := make(map[int64]bool)   // telegramID -> isDeleted
	for _, reg := range existingRegs {
		existingMap[reg.User.TelegramID] = reg.UserID
		// Проверяем, является ли запись soft-deleted
		if reg.DeletedAt.Valid {
			deletedMap[reg.User.TelegramID] = true
		}
	}

	for telegramID, reg := range registrations {
		var user models.UserGORM
		if err := r.db.WithContext(ctx).
			Where("telegram_id = ?", telegramID).
			First(&user).Error; err != nil {
			continue
		}

		regModel := &models.EventRegistrationGORM{
			EventID:   string(eventID),
			UserID:    user.ID,
			Status:    string(reg.Status),
			CreatedAt: reg.CreatedAt,
			UpdatedAt: reg.UpdatedAt,
		}

		if userID, exists := existingMap[telegramID]; exists {
			// Запись существует (возможно, soft-deleted)
			if deletedMap[telegramID] {
				// Восстанавливаем soft-deleted запись
				if err := r.db.WithContext(ctx).Unscoped().
					Model(&models.EventRegistrationGORM{}).
					Where("event_id = ? AND user_id = ?", string(eventID), userID).
					Updates(map[string]interface{}{
						"status":     regModel.Status,
						"updated_at": regModel.UpdatedAt,
						"deleted_at": nil, // Восстанавливаем запись
					}).Error; err != nil {
					return err
				}
			} else {
				// Обновляем существующую запись
				if err := r.db.WithContext(ctx).
					Model(&models.EventRegistrationGORM{}).
					Where("event_id = ? AND user_id = ?", string(eventID), userID).
					Updates(regModel).Error; err != nil {
					return err
				}
			}
		} else {
			// Создаем новую запись
			if err := r.db.WithContext(ctx).Create(regModel).Error; err != nil {
				return err
			}
		}
	}

	var telegramIDsToKeep []int64
	for telegramID := range registrations {
		telegramIDsToKeep = append(telegramIDsToKeep, telegramID)
	}

	if len(telegramIDsToKeep) > 0 {
		var users []models.UserGORM
		if err := r.db.WithContext(ctx).
			Where("telegram_id IN ?", telegramIDsToKeep).
			Find(&users).Error; err != nil {
			return err
		}

		var userIDsToKeep []int64
		for _, user := range users {
			userIDsToKeep = append(userIDsToKeep, user.ID)
		}

		if len(userIDsToKeep) > 0 {
			if err := r.db.WithContext(ctx).
				Where("event_id = ? AND user_id NOT IN ?", string(eventID), userIDsToKeep).
				Delete(&models.EventRegistrationGORM{}).Error; err != nil {
				return err
			}
		}
	} else {
		if err := r.db.WithContext(ctx).
			Where("event_id = ?", string(eventID)).
			Delete(&models.EventRegistrationGORM{}).Error; err != nil {
			return err
		}
	}

	return nil
}
