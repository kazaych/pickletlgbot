package location

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Service сервис для работы с локациями (бизнес-логика)
// Не знает ни про Redis, ни про Telegram
type Service struct {
	repo Repository
}

// NewService создает новый сервис для работы с локациями
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// ListLocations получает список всех локаций
func (s *Service) ListLocations(ctx context.Context) ([]*Location, error) {
	scanner, err := s.repo.ScanLocations(ctx, "location*")
	if err != nil {
		return nil, err
	}
	defer scanner.Close()

	count, err := scanner.GetLocations()
	if err != nil {
		return nil, err
	}

	var locations []*Location
	for i := 0; i < count; i++ {
		location := scanner.GetLocation(i)
		if location != nil {
			locations = append(locations, location)
		}
	}

	return locations, nil
}

// GetLocation получает информацию о локации по ID
func (s *Service) GetLocation(ctx context.Context, locationID string) (*Location, error) {
	location, err := s.repo.GetLocation(ctx, locationID)
	if err != nil {
		return nil, fmt.Errorf("локация не найдена: %w", err)
	}

	return location, nil
}

// CreateLocation создает новую локацию
func (s *Service) CreateLocation(ctx context.Context, name string, address string, mapUrl string) (*Location, error) {
	locationID := uuid.New().String()

	location := &Location{
		ID:            locationID,
		Name:          name,
		Address:       address,
		AddressMapUrl: mapUrl,
	}

	err := s.repo.CreateLocation(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания локации: %w", err)
	}

	return location, nil
}

// DeleteLocation удаляет локацию
func (s *Service) DeleteLocation(ctx context.Context, locationID string) error {
	return s.repo.DeleteLocation(ctx, locationID)
}

// UpdateLocation обновляет локацию
func (s *Service) UpdateLocation(ctx context.Context, location *Location) error {
	return s.repo.UpdateLocation(ctx, location)
}
