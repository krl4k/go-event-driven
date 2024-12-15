package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"strconv"
	"tickets/internal/entities"
	"time"
)

// Ticket represents the ticket entity
type Ticket struct {
	ID            uuid.UUID  `db:"ticket_id"`
	PriceAmount   float64    `db:"price_amount"`
	PriceCurrency string     `db:"price_currency"`
	CustomerEmail string     `db:"customer_email"`
	DeletedAt     *time.Time `db:"deleted_at"`
}

type TicketsRepo struct {
	db *sqlx.DB
}

func NewTicketsRepo(db *sqlx.DB) *TicketsRepo {
	return &TicketsRepo{db: db}
}

func (r *TicketsRepo) Create(ctx context.Context, t *entities.Ticket) error {
	query := `
        INSERT INTO tickets (
            ticket_id, price_amount, price_currency, customer_email
        ) VALUES (
            $1, $2, $3, $4
        ) ON CONFLICT DO NOTHING`

	ticket, err := domainToModel(t)
	if err != nil {
		return fmt.Errorf("failed to convert entities to model: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		ticket.ID,
		ticket.PriceAmount,
		ticket.PriceCurrency,
		ticket.CustomerEmail,
	)
	return err
}

var ErrTicketNotFound = fmt.Errorf("ticket not found")

func (r *TicketsRepo) Delete(ctx context.Context, ticketID uuid.UUID) error {
	query := `
		UPDATE tickets SET deleted_at = $1 WHERE ticket_id = $2`

	res, err := r.db.ExecContext(ctx, query, time.Now().UTC(), ticketID)
	if err != nil {
		return fmt.Errorf("failed to delete ticket: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", ErrTicketNotFound)
	}

	if rowsAffected == 0 {
		return ErrTicketNotFound
	}
	return err
}

func (r *TicketsRepo) List(ctx context.Context) ([]entities.Ticket, error) {
	var tickets []Ticket
	query := `
		SELECT ticket_id, price_amount, price_currency, customer_email
		FROM tickets
		WHERE deleted_at IS NULL`

	err := r.db.SelectContext(ctx, &tickets, query)
	if err != nil {
		return nil, err
	}

	convertedTickets := make([]entities.Ticket, 0, len(tickets))
	for _, ticket := range tickets {
		convertedTickets = append(convertedTickets, modelToDomain(ticket))
	}

	return convertedTickets, nil
}

func modelToDomain(ticket Ticket) entities.Ticket {
	return entities.Ticket{
		TicketId:      ticket.ID.String(),
		CustomerEmail: ticket.CustomerEmail,
		Price: entities.Money{
			Amount:   strconv.FormatFloat(ticket.PriceAmount, 'f', 2, 64),
			Currency: ticket.PriceCurrency,
		},
	}
}

func domainToModel(ticket *entities.Ticket) (*Ticket, error) {
	id, err := uuid.Parse(ticket.TicketId)
	if err != nil {
		return nil, err
	}

	amount, err := strconv.ParseFloat(ticket.Price.Amount, 64)
	if err != nil {
		return nil, err
	}

	return &Ticket{
		ID:            id,
		PriceAmount:   amount,
		PriceCurrency: ticket.Price.Currency,
		CustomerEmail: ticket.CustomerEmail,
	}, nil
}
