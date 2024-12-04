package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domain "tickets/internal/domain/shows"
)

type ShowsRepo struct {
	db *sqlx.DB
}

func NewShowsRepo(db *sqlx.DB) *ShowsRepo {
	return &ShowsRepo{db: db}
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

	err := r.db.QueryRowContext(ctx, query,
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
