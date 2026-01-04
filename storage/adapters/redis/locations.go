package redis

import (
	"context"
	"fmt"

	"kitchenBot/domain/location"
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
	key := fmt.Sprintf("location:%s", loc.ID)

	// Проверяем, не существует ли уже локация с таким ID
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists > 0 {
		return fmt.Errorf("локация с ID %s уже существует", loc.ID)
	}

	// Сохраняем локацию как Hash
	fields := map[string]interface{}{
		"id":            loc.ID,
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
func (r *RedisLocationRepository) DeleteLocation(ctx context.Context, locationID string) error {
	key := fmt.Sprintf("location:%s", locationID)

	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return err
	}

	return nil
}

// UpdateLocation обновляет локацию
func (r *RedisLocationRepository) UpdateLocation(ctx context.Context, loc *location.Location) error {
	key := fmt.Sprintf("location:%s", loc.ID)

	// Проверяем, существует ли локация
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return fmt.Errorf("локация с ID %s не найдена", loc.ID)
	}

	// Обновляем поля локации в Hash
	fields := map[string]interface{}{
		"id":            loc.ID,
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
func (r *RedisLocationRepository) GetLocation(ctx context.Context, locationID string) (*location.Location, error) {
	key := fmt.Sprintf("location:%s", locationID)

	// Получаем все поля Hash
	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	// Проверяем, что локация существует
	if len(result) == 0 {
		return nil, fmt.Errorf("локация с ID %s не найдена", locationID)
	}

	// Создаем локацию из полей Hash
	loc := &location.Location{
		ID:            result["id"],
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
