package repository

import (
	"context"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domain "tickets/internal/domain/bookings"
)

type BookingsRepo struct {
	db *sqlx.DB
}

func NewBookingsRepo(db *sqlx.DB) *BookingsRepo {
	return &BookingsRepo{db: db}
}

func (r *BookingsRepo) CreateBooking(ctx context.Context, booking domain.Booking) (uuid.UUID, error) {
	var id uuid.UUID

	query := `
		INSERT INTO bookings (
			show_id, number_of_tickets, customer_email
		) VALUES (
			$1, $2, $3
		) RETURNING id`

	err := r.db.QueryRowContext(ctx, query,
		booking.ShowId,
		booking.NumberOfTickets,
		booking.CustomerEmail,
	).Scan(&id)

	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}
