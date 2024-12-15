package shows

import (
	"context"
	"github.com/google/uuid"
	"tickets/internal/entities"
)

//go:generate mockgen -destination=../mocks/mock_shows_service.go -package=mocks tickets/internal/application/services ShowsService
type ShowsRepo interface {
	CreateShow(ctx context.Context, show entities.Show) (uuid.UUID, error)
}

type CreateShowUsecase struct {
	showsRepo ShowsRepo
}

func NewShowsService(showsRepo ShowsRepo) *CreateShowUsecase {
	return &CreateShowUsecase{
		showsRepo: showsRepo,
	}
}

func (s *CreateShowUsecase) CreateShow(ctx context.Context, show entities.Show) (uuid.UUID, error) {
	return s.showsRepo.CreateShow(ctx, show)
}
