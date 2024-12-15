package repository

import (
	"context"
	"fmt"
	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"tickets/internal/entities"
)

type BookingsRepo struct {
	db     *sqlx.DB
	getter *trmsqlx.CtxGetter
}

func NewBookingsRepo(
	db *sqlx.DB,
	getter *trmsqlx.CtxGetter,
) *BookingsRepo {
	return &BookingsRepo{
		db:     db,
		getter: getter,
	}
}

func (r *BookingsRepo) CreateBooking(ctx context.Context, booking entities.Booking) (uuid.UUID, error) {
	var id uuid.UUID

	query := `
		INSERT INTO bookings (
			show_id, number_of_tickets, customer_email
		) VALUES (
			$1, $2, $3
		) RETURNING id`

	err := r.getter.DefaultTrOrDB(ctx, r.db).
		QueryRowContext(ctx, query,
			booking.ShowId,
			booking.NumberOfTickets,
			booking.CustomerEmail,
		).Scan(&id)

	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (r *BookingsRepo) GetBookingsCountByShowID(ctx context.Context, showID uuid.UUID) (int64, error) {
	var count int64

	// sum number of tickets for all bookings for the show
	query := `
		SELECT COALESCE(SUM(number_of_tickets), 0)
		FROM bookings
		WHERE show_id = $1`

	err := r.getter.DefaultTrOrDB(ctx, r.db).
		QueryRowContext(ctx, query, showID).
		Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get bookings count: %w", err)
	}

	return count, nil
}
