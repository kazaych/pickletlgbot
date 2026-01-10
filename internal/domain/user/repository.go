package user

import "context"

type UserRepository interface {
	Save(ctx context.Context, player *User) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (*User, error)
	ListByEventID(ctx context.Context, eventID int64) ([]User, error)
	ListByLocationID(ctx context.Context, locationID int64) ([]User, error)
}
