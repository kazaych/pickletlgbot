package redis

import (
	"context"
	"fmt"

	"kitchenBot/domain/location"

	"github.com/google/uuid"
)

// RedisLocationRepository реализация LocationRepository для Redis
type RedisLocationRepository struct {
	client *Client
}

// NewLocationRepository создает новый репозиторий для работы с локациями
func NewLocationRepository(client *Client) location.Repository {
	return &RedisLocationRepository{client: client}
}

// CreateLocation создает новую локацию в Redis
// Формат ключа: location:{id}
// Используется Redis Hash для хранения полей локации
func (r *RedisLocationRepository) CreateLocation(ctx context.Context, loc *location.Location) error {
	key := fmt.Sprintf("location:%s", loc.ID.String())

	// Проверяем, не существует ли уже локация с таким ID
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists > 0 {
		return fmt.Errorf("локация с ID %s уже существует", loc.ID.String())
	}

	// Сохраняем локацию как Hash
	fields := map[string]interface{}{
		"id":            loc.ID.String(),
		"name":          loc.Name,
		"address":       loc.Address,
		"addressMapUrl": loc.AddressMapUrl,
	}

	err = r.client.HSet(ctx, key, fields).Err()
	if err != nil {
		return err
	}

	return nil
}

// DeleteLocation удаляет локацию из Redis
func (r *RedisLocationRepository) DeleteLocation(ctx context.Context, locationID uuid.UUID) error {
	key := fmt.Sprintf("location:%s", locationID.String())

	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return err
	}

	return nil
}

// UpdateLocation обновляет локацию
func (r *RedisLocationRepository) UpdateLocation(ctx context.Context, loc *location.Location) error {
	key := fmt.Sprintf("location:%s", loc.ID.String())

	// Проверяем, существует ли локация
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return fmt.Errorf("локация с ID %s не найдена", loc.ID.String())
	}

	// Обновляем поля локации в Hash
	fields := map[string]interface{}{
		"id":            loc.ID.String(),
		"name":          loc.Name,
		"address":       loc.Address,
		"addressMapUrl": loc.AddressMapUrl,
	}

	err = r.client.HSet(ctx, key, fields).Err()
	if err != nil {
		return err
	}

	return nil
}

// GetLocation получает локацию по ID
func (r *RedisLocationRepository) GetLocation(ctx context.Context, locationID uuid.UUID) (*location.Location, error) {
	key := fmt.Sprintf("location:%s", locationID.String())

	// Получаем все поля Hash
	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	// Проверяем, что локация существует
	if len(result) == 0 {
		return nil, fmt.Errorf("локация с ID %s не найдена", locationID.String())
	}

	// Парсим ID
	var id uuid.UUID
	if idStr, ok := result["id"]; ok && idStr != "" {
		id, err = uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("неверный формат ID локации: %w", err)
		}
	} else {
		// Если ID не найден в Hash, используем переданный locationID
		id = locationID
	}

	// Создаем локацию из полей Hash
	loc := &location.Location{
		ID:            id,
		Name:          result["name"],
		Address:       result["address"],
		AddressMapUrl: result["addressMapUrl"],
	}

	return loc, nil
}

// ScanLocations сканирует локации по маске
func (r *RedisLocationRepository) ScanLocations(ctx context.Context, mask string) (location.Scanner, error) {
	return NewScanResult(ctx, r.client, mask), nil
}
