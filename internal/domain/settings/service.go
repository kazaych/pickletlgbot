package settings

import (
	"context"
	"strconv"
)

type Service interface {
	GetChannelID(ctx context.Context) (int64, error)
	SetChannelID(ctx context.Context, channelID int64) error
}

type settingsService struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &settingsService{repo: repo}
}

func (s *settingsService) GetChannelID(ctx context.Context) (int64, error) {
	val, err := s.repo.Get(ctx, KeyChannelID)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

func (s *settingsService) SetChannelID(ctx context.Context, channelID int64) error {
	return s.repo.Set(ctx, KeyChannelID, strconv.FormatInt(channelID, 10))
}
