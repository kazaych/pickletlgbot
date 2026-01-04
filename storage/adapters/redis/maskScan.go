package redis

import (
	"context"
	"strings"

	"kitchenBot/domain/location"
)

// ScanResult реализация LocationScanner для Redis
type ScanResult struct {
	Locations   []*location.Location
	Cursor      uint64
	ScanMask    string
	RedisClient *Client
	ctx         context.Context
}

func NewScanResult(ctx context.Context, client *Client, mask string) *ScanResult {
	return &ScanResult{
		Locations:   []*location.Location{},
		Cursor:      0,
		ScanMask:    mask,
		RedisClient: client,
		ctx:         ctx,
	}
}

func (sr *ScanResult) GetLocations() (int, error) {
	for {
		res, currentCursor, err := sr.RedisClient.Scan(sr.ctx, sr.Cursor, sr.ScanMask, 100).Result()
		if err != nil {
			return 0, err
		}
		if len(res) != 0 {
			for _, key := range res {
				// Проверяем, что ключ соответствует формату location:{id}
				if !strings.HasPrefix(key, "location:") {
					continue
				}

				// Получаем все поля Hash
				result, err := sr.RedisClient.HGetAll(sr.ctx, key).Result()
				if err != nil {
					// Если не удалось получить данные, пропускаем этот ключ
					continue
				}

				// Проверяем, что Hash не пустой
				if len(result) == 0 {
					continue
				}

				// Создаем локацию из полей Hash
				loc := &location.Location{
					ID:            result["id"],
					Name:          result["name"],
					Address:       result["address"],
					AddressMapUrl: result["addressMapUrl"],
				}

				// Если ID не был сохранен в Hash, извлекаем из ключа
				if loc.ID == "" {
					if len(key) > 9 && key[:9] == "location:" {
						loc.ID = key[9:]
					}
				}

				sr.Locations = append(sr.Locations, loc)
			}
		}
		sr.Cursor = currentCursor
		if sr.Cursor == 0 {
			break
		}
	}

	return len(sr.Locations), nil
}

// GetLocation возвращает локацию по индексу
func (sr *ScanResult) GetLocation(index int) *location.Location {
	if index < 0 || index >= len(sr.Locations) {
		return nil
	}
	return sr.Locations[index]
}

// Close закрывает соединение с Redis
func (sr *ScanResult) Close() error {
	// В новой архитектуре клиент управляется извне, поэтому здесь ничего не закрываем
	return nil
}
