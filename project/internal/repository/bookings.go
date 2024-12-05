package repository

import (
	"context"
	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domain "tickets/internal/domain/bookings"
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

func (r *BookingsRepo) CreateBooking(ctx context.Context, booking domain.Booking) (uuid.UUID, error) {
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
