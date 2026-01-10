package postgres

import (
	"context"
	"errors"
	"pickletlgbot/internal/domain/user"
	"pickletlgbot/internal/models"

	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) user.UserRepository {
	return &userRepository{db: db}
}

func (ur *userRepository) Save(ctx context.Context, usr *user.User) error {
	model := &models.UserGORM{
		ID:         usr.ID,
		Name:       usr.Name,
		Surname:    usr.Surname,
		TelegramID: usr.TelegramID,
	}

	// Если ID = 0, создаем новую запись, иначе обновляем существующую
	if usr.ID == 0 {
		if err := ur.db.WithContext(ctx).Create(model).Error; err != nil {
			return err
		}
		// Обновляем ID в доменной модели после создания
		usr.ID = model.ID
	} else {
		if err := ur.db.WithContext(ctx).
			Model(&models.UserGORM{}).
			Where("id = ?", usr.ID).
			Updates(model).Error; err != nil {
			return err
		}
	}

	return nil
}

func (ur *userRepository) Delete(ctx context.Context, id int64) error {
	return ur.db.WithContext(ctx).Delete(&models.UserGORM{}, id).Error
}

func (ur *userRepository) GetByID(ctx context.Context, id int64) (*user.User, error) {
	var model models.UserGORM
	if err := ur.db.WithContext(ctx).
		Where("id = ?", id).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return ur.modelToDomain(&model), nil
}

func (ur *userRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*user.User, error) {
	var model models.UserGORM
	if err := ur.db.WithContext(ctx).
		Where("telegram_id = ?", telegramID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return ur.modelToDomain(&model), nil
}

func (ur *userRepository) ListByEventID(ctx context.Context, eventID int64) ([]user.User, error) {
	// Получаем пользователей через JOIN с event_registrations
	// Используем Preload для загрузки связей через GORM
	var registrations []models.EventRegistrationGORM
	if err := ur.db.WithContext(ctx).
		Preload("User").
		Where("event_id = ? AND deleted_at IS NULL", eventID).
		Find(&registrations).Error; err != nil {
		return nil, err
	}

	users := make([]user.User, 0, len(registrations))
	for _, reg := range registrations {
		if reg.User.ID != 0 { // Проверяем, что пользователь загружен
			users = append(users, *ur.modelToDomain(&reg.User))
		}
	}
	return users, nil
}

func (ur *userRepository) ListByLocationID(ctx context.Context, locationID int64) ([]user.User, error) {
	// Получаем пользователей через JOIN с events и event_registrations
	// Сначала находим события по location_id, потом пользователей через регистрации
	var events []models.EventGORM
	if err := ur.db.WithContext(ctx).
		Where("location_id = ? AND deleted_at IS NULL", locationID).
		Find(&events).Error; err != nil {
		return nil, err
	}

	// Собираем уникальных пользователей из всех событий
	userMap := make(map[int64]*models.UserGORM)
	for _, evt := range events {
		var registrations []models.EventRegistrationGORM
		if err := ur.db.WithContext(ctx).
			Preload("User").
			Where("event_id = ? AND deleted_at IS NULL", evt.EventID).
			Find(&registrations).Error; err != nil {
			continue
		}

		for _, reg := range registrations {
			if reg.User.ID != 0 {
				userMap[reg.User.ID] = &reg.User
			}
		}
	}

	users := make([]user.User, 0, len(userMap))
	for _, model := range userMap {
		users = append(users, *ur.modelToDomain(model))
	}
	return users, nil
}

// modelToDomain конвертирует GORM модель в доменную модель
func (ur *userRepository) modelToDomain(model *models.UserGORM) *user.User {
	return &user.User{
		ID:         model.ID,
		Name:       model.Name,
		Surname:    model.Surname,
		TelegramID: model.TelegramID,
	}
}
