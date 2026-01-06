package location

import "context"

// LocationRepository описывает, что нужно домену от хранилища локаций.
type LocationRepository interface {
	// GetByID возвращает локацию по ID или ошибку, если не найдено/сломалось хранилище.
	GetByID(ctx context.Context, id LocationID) (*Location, error)

	// List возвращает все доступные локации (для выбора в боте).
	List(ctx context.Context) ([]Location, error)

	// Save создаёт или обновляет локацию.
	Save(ctx context.Context, loc *Location) error

	// Delete удаляет локацию по ID (если пригодится админский функционал).
	Delete(ctx context.Context, id LocationID) error
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
