package settings

import (
	"context"
	"strconv"
	"strings"
)

type Service interface {
	GetChannelIDs(ctx context.Context) ([]int64, error)
	AddChannelID(ctx context.Context, channelID int64) error
	RemoveChannelID(ctx context.Context, channelID int64) error
}

type settingsService struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &settingsService{repo: repo}
}

func (s *settingsService) GetChannelIDs(ctx context.Context) ([]int64, error) {
	val, err := s.repo.Get(ctx, KeyChannelIDs)
	if err != nil || val == "" {
		return nil, err
	}
	var ids []int64
	for _, part := range strings.Split(val, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (s *settingsService) AddChannelID(ctx context.Context, channelID int64) error {
	existing, err := s.GetChannelIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range existing {
		if id == channelID {
			return nil // уже есть
		}
	}
	existing = append(existing, channelID)
	return s.repo.Set(ctx, KeyChannelIDs, joinIDs(existing))
}

func (s *settingsService) RemoveChannelID(ctx context.Context, channelID int64) error {
	existing, err := s.GetChannelIDs(ctx)
	if err != nil {
		return err
	}
	var filtered []int64
	for _, id := range existing {
		if id != channelID {
			filtered = append(filtered, id)
		}
	}
	return s.repo.Set(ctx, KeyChannelIDs, joinIDs(filtered))
}

func joinIDs(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return strings.Join(parts, ",")
}
