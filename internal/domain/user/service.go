package user

import "context"

type UserService interface {
	CreateUser(ctx context.Context, player *User) error
	DeleteUser(ctx context.Context, id int64) error
	IsUserExists(ctx context.Context, telegramID int64) (bool, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (*User, error)
}

type userService struct {
	repository UserRepository
}

func NewPlayerService(repository UserRepository) UserService {
	return &userService{repository: repository}
}

func (ps *userService) CreateUser(ctx context.Context, player *User) error {
	// Проверяем, существует ли пользователь
	exists, err := ps.IsUserExists(ctx, player.TelegramID)
	if err != nil {
		return err
	}

	// Если пользователь уже существует, обновляем его данные
	if exists {
		existingUser, err := ps.repository.GetByTelegramID(ctx, player.TelegramID)
		if err != nil {
			return err
		}
		if existingUser != nil {
			// Обновляем существующего пользователя
			existingUser.Name = player.Name
			existingUser.Surname = player.Surname
			return ps.repository.Save(ctx, existingUser)
		}
	}

	// Создаем нового пользователя
	return ps.repository.Save(ctx, player)
}

func (ps *userService) DeleteUser(ctx context.Context, id int64) error {
	return ps.repository.Delete(ctx, id)
}

func (ps *userService) IsUserExists(ctx context.Context, telegramID int64) (bool, error) {
	player, err := ps.repository.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return false, err
	}
	return player != nil, nil
}

func (ps *userService) GetByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	return ps.repository.GetByTelegramID(ctx, telegramID)
}
