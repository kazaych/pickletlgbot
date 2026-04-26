package postgres

import (
	"context"
	"errors"
	"pickletlgbot/internal/models"

	"gorm.io/gorm"
)

type settingsRepository struct {
	db *gorm.DB
}

func NewSettingsRepository(db *gorm.DB) *settingsRepository {
	return &settingsRepository{db: db}
}

func (r *settingsRepository) Get(ctx context.Context, key string) (string, error) {
	var s models.SettingsGORM
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return s.Value, nil
}

func (r *settingsRepository) Set(ctx context.Context, key, value string) error {
	var s models.SettingsGORM
	return r.db.WithContext(ctx).
		Where("key = ?", key).
		Assign(models.SettingsGORM{Key: key, Value: value}).
		FirstOrCreate(&s).Error
}
