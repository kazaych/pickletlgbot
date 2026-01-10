package location

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// LocationService описывает use-case'ы вокруг локаций.
type LocationService interface {
	Get(ctx context.Context, id LocationID) (*Location, error)
	List(ctx context.Context) ([]Location, error)
	Create(ctx context.Context, input CreateLocationInput) (*Location, error)
	Update(ctx context.Context, id LocationID, input UpdateLocationInput) (*Location, error)
	Delete(ctx context.Context, id LocationID) error
}

// DTO для создания/обновления — чтобы не таскать всю структуру.
type CreateLocationInput struct {
	Name          string
	Address       string
	Description   string
	AddressMapURL string
}

type UpdateLocationInput struct {
	Name          *string
	Address       *string
	Description   *string
	AddressMapURL *string
}

type locationService struct {
	repo LocationRepository
}

func NewService(repo LocationRepository) LocationService {
	return &locationService{repo: repo}
}

func (s *locationService) Get(ctx context.Context, id LocationID) (*Location, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *locationService) List(ctx context.Context) ([]Location, error) {
	return s.repo.List(ctx)
}

func (s *locationService) Create(ctx context.Context, in CreateLocationInput) (*Location, error) {
	loc := &Location{
		ID:            generateID(),
		Name:          in.Name,
		Address:       in.Address,
		Description:   in.Description,
		AddressMapURL: in.AddressMapURL,
	}
	if len(loc.Name) == 0 || len(loc.Address) == 0 {
		return nil, errors.New("name and address are required")
	}

	if err := s.repo.Save(ctx, loc); err != nil {
		return nil, err
	}
	return loc, nil
}

func (s *locationService) Update(
	ctx context.Context,
	id LocationID,
	in UpdateLocationInput,
) (*Location, error) {
	loc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		loc.Name = *in.Name
	}
	if in.Address != nil {
		loc.Address = *in.Address
	}
	if in.Description != nil {
		loc.Description = *in.Description
	}
	if in.AddressMapURL != nil {
		loc.AddressMapURL = *in.AddressMapURL
	}

	// тут тоже можно проверять инварианты (например, длину строк)

	if err := s.repo.Save(ctx, loc); err != nil {
		return nil, err
	}
	return loc, nil
}

func (s *locationService) Delete(ctx context.Context, id LocationID) error {
	return s.repo.Delete(ctx, id)
}

func generateID() LocationID {
	return LocationID(uuid.New().String())
}
