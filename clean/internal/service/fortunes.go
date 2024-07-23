package service

import (
	"context"
	"sort"

	"github.com/anton2920/techempower/clean/internal/entity"
	"github.com/anton2920/techempower/clean/internal/repository"
)

type FortunesService interface {
	GetAllSorted(context.Context) ([]entity.Fortune, error)
}

type fortunesService struct {
	repo repository.FortunesRepository
}

func NewFortunesService(r repository.FortunesRepository) FortunesService {
	return &fortunesService{
		repo: r,
	}
}

func (s *fortunesService) GetAllSorted(ctx context.Context) ([]entity.Fortune, error) {
	fortunes, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	fortunes = append(fortunes, entity.Fortune{ID: len(fortunes), Message: "Additional fortune added at request time."})

	sort.Slice(fortunes, func(i, j int) bool {
		return fortunes[i].Message < fortunes[j].Message
	})

	return fortunes, nil
}
