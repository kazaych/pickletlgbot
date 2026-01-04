package location

import (
	"context"

	"github.com/google/uuid"
)

// Repository интерфейс для работы с локациями в хранилище
type Repository interface {
	// CreateLocation создает новую локацию
	CreateLocation(ctx context.Context, location *Location) error

	// DeleteLocation удаляет локацию
	DeleteLocation(ctx context.Context, locationID uuid.UUID) error

	// UpdateLocation обновляет локацию
	UpdateLocation(ctx context.Context, location *Location) error

	// GetLocation получает локацию по ID
	GetLocation(ctx context.Context, locationID uuid.UUID) (*Location, error)

	// ScanLocations сканирует локации по маске
	ScanLocations(ctx context.Context, mask string) (Scanner, error)
}

// Scanner интерфейс для сканирования локаций
type Scanner interface {
	// GetLocations получает все локации
	GetLocations() (int, error)

	// GetLocation возвращает локацию по индексу
	GetLocation(index int) *Location

	// Close закрывает сканер
	Close() error
}
