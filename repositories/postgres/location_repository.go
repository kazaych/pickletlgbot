package postgres

import (
	"context"
	"errors"

	"pickletlgbot/internal/domain/location"
	"pickletlgbot/internal/models"

	"gorm.io/gorm"
)

type locationRepository struct {
	db *gorm.DB
}

func NewLocationRepository(db *gorm.DB) location.LocationRepository {
	return &locationRepository{db: db}
}

func (r *locationRepository) GetByID(ctx context.Context, id location.LocationID) (*location.Location, error) {
	var model models.LocationGORM
	if err := r.db.WithContext(ctx).
		Where("location_id = ?", id).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // или custom ErrNotFound
		}
		return nil, err
	}

	return &location.Location{
		ID:            location.LocationID(model.LocationID),
		Name:          model.Name,
		Address:       model.Address,
		Description:   model.Description,
		AddressMapURL: model.AddressMapURL,
	}, nil
}

func (r *locationRepository) List(ctx context.Context) ([]location.Location, error) {
	var models []models.LocationGORM
	if err := r.db.WithContext(ctx).
		Unscoped(). // показывать удалённые? Или Where("deleted_at IS NULL")
		Find(&models).Error; err != nil {
		return nil, err
	}

	locations := make([]location.Location, len(models))
	for i, m := range models {
		locations[i] = location.Location{
			ID:            location.LocationID(m.LocationID),
			Name:          m.Name,
			Address:       m.Address,
			Description:   m.Description,
			AddressMapURL: m.AddressMapURL,
		}
	}
	return locations, nil
}

func (r *locationRepository) Save(ctx context.Context, loc *location.Location) error {
	model := &models.LocationGORM{
		LocationID:    string(loc.ID),
		Name:          loc.Name,
		Address:       loc.Address,
		Description:   loc.Description,
		AddressMapURL: loc.AddressMapURL,
	}

	return r.db.WithContext(ctx).
		Where("location_id = ?", loc.ID).
		Assign(model).
		FirstOrCreate(model).Error
}

func (r *locationRepository) Delete(ctx context.Context, id location.LocationID) error {
	return r.db.WithContext(ctx).
		Where("location_id = ?", id).
		Delete(&models.LocationGORM{}).Error
}
