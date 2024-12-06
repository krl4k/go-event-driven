package repository

import (
	"context"
	"fmt"
	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domain "tickets/internal/domain/shows"
)

type ShowsRepo struct {
	db     *sqlx.DB
	getter *trmsqlx.CtxGetter
}

func NewShowsRepo(
	db *sqlx.DB,
	getter *trmsqlx.CtxGetter,
) *ShowsRepo {
	return &ShowsRepo{
		db:     db,
		getter: getter,
	}
}

func (r *ShowsRepo) CreateShow(ctx context.Context, show domain.Show) (uuid.UUID, error) {
	var id uuid.UUID

	query := `
       INSERT INTO shows (
          dead_nation_id, number_of_tickets, start_time, title, venue
       ) VALUES (
          $1, $2, $3, $4, $5
       ) ON CONFLICT DO NOTHING
       RETURNING id`

	err := r.getter.DefaultTrOrDB(ctx, r.db).QueryRowContext(ctx, query,
		show.DeadNationId,
		show.NumberOfTickets,
		show.StartTime,
		show.Title,
		show.Venue,
	).Scan(&id)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create show: %w", err)
	}

	return id, nil
}

func (r *ShowsRepo) GetShow(ctx context.Context, id uuid.UUID) (*domain.Show, error) {
	var show domain.Show

	query := `
	   SELECT
		  id, dead_nation_id, number_of_tickets, start_time, title, venue
	   FROM shows
	   WHERE id = $1`

	err := r.getter.DefaultTrOrDB(ctx, r.db).QueryRowContext(ctx, query, id).
		Scan(&show.Id, &show.DeadNationId, &show.NumberOfTickets, &show.StartTime, &show.Title, &show.Venue)

	if err != nil {
		return nil, fmt.Errorf("failed to get show: %w", err)
	}

	return &show, nil
}
