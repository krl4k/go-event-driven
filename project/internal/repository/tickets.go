package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"strconv"
	domain "tickets/internal/domain/tickets"
)

// Ticket represents the ticket entity
type Ticket struct {
	ID            uuid.UUID `db:"ticket_id"`
	PriceAmount   float64   `db:"price_amount"`
	PriceCurrency string    `db:"price_currency"`
	CustomerEmail string    `db:"customer_email"`
}

type TicketsRepo struct {
	db *sqlx.DB
}

func NewTicketsRepo(db *sqlx.DB) *TicketsRepo {
	return &TicketsRepo{db: db}
}

// Create inserts a new ticket record
func (r *TicketsRepo) Create(ctx context.Context, t *domain.Ticket) error {
	query := `
        INSERT INTO tickets (
            ticket_id, price_amount, price_currency, customer_email
        ) VALUES (
            $1, $2, $3, $4
        )`

	ticket, err := domainToModel(t)
	if err != nil {
		return fmt.Errorf("failed to convert domain to model: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		ticket.ID,
		ticket.PriceAmount,
		ticket.PriceCurrency,
		ticket.CustomerEmail,
	)
	return err
}

func domainToModel(ticket *domain.Ticket) (*Ticket, error) {
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

func (r *TicketsRepo) Delete(ctx context.Context, ticketID uuid.UUID) error {
	query := `DELETE FROM tickets WHERE ticket_id = $1`
	_, err := r.db.ExecContext(ctx, query, ticketID)
	return err
}

func (r *TicketsRepo) List(ctx context.Context) ([]domain.Ticket, error) {
	var tickets []Ticket
	query := `
		SELECT ticket_id, price_amount, price_currency, customer_email
		FROM tickets`

	err := r.db.SelectContext(ctx, &tickets, query)
	if err != nil {
		return nil, err
	}

	convertedTickets := make([]domain.Ticket, 0, len(tickets))
	for _, ticket := range tickets {
		convertedTickets = append(convertedTickets, modelToDomain(ticket))
	}

	return convertedTickets, nil
}

func modelToDomain(ticket Ticket) domain.Ticket {
	return domain.Ticket{
		TicketId:      ticket.ID.String(),
		CustomerEmail: ticket.CustomerEmail,
		Price: domain.Money{
			Amount:   strconv.FormatFloat(ticket.PriceAmount, 'f', 2, 64),
			Currency: ticket.PriceCurrency,
		},
	}
}
