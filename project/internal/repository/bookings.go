package repository

import (
	"context"
	"errors"
	"fmt"
	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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

const (
	postgresUniqueValueViolationErrorCode = "23505"
)

func isErrorUniqueViolation(err error) bool {
	var psqlErr *pq.Error
	return errors.As(err, &psqlErr) && psqlErr.Code == postgresUniqueValueViolationErrorCode
}

var ErrBookingAlreadyExists = errors.New("booking already exists")

func (r *BookingsRepo) CreateBooking(ctx context.Context, booking entities.Booking) (uuid.UUID, error) {
	query := `
		INSERT INTO bookings (
			id, show_id, number_of_tickets, customer_email
		) VALUES (
			$1, $2, $3, $4
		)`

	res, err := r.getter.DefaultTrOrDB(ctx, r.db).
		ExecContext(ctx, query,
			booking.Id,
			booking.ShowId,
			booking.NumberOfTickets,
			booking.CustomerEmail,
		)

	if err != nil {
		if isErrorUniqueViolation(err) {
			// deduplication
			return uuid.Nil, ErrBookingAlreadyExists
		}
		return uuid.Nil, fmt.Errorf("insert booking: %w", err)
	}
	count, err := res.RowsAffected()
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get rows affected: %w", err)
	}
	if count == 0 {
		return uuid.Nil, fmt.Errorf("no rows affected")
	}

	return booking.Id, nil
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
