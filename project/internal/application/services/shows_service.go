package services

import (
	"context"
	"github.com/google/uuid"
	domain "tickets/internal/domain/shows"
)

//go:generate mockgen -destination=../mocks/mock_shows_service.go -package=mocks tickets/internal/application/services ShowsService
type ShowsRepo interface {
	CreateShow(ctx context.Context, show domain.Show) (uuid.UUID, error)
}

type ShowsService struct {
	showsRepo ShowsRepo
}

func NewShowsService(showsRepo ShowsRepo) *ShowsService {
	return &ShowsService{
		showsRepo: showsRepo,
	}
}

func (s *ShowsService) CreateShow(ctx context.Context, show domain.Show) (uuid.UUID, error) {
	return s.showsRepo.CreateShow(ctx, show)
}
