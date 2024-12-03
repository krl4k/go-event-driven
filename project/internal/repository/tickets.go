package repository

import (
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
func (r *TicketsRepo) Create(t *domain.Ticket) error {
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

	_, err = r.db.Exec(query,
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

//func (r *TicketsRepo) GetByID(id uuid.UUID) (*Ticket, error) {
//	var ticket Ticket
//	query := `
//        SELECT ticket_id, price_amount, price_currency, customer_email
//        FROM tickets
//        WHERE ticket_id = $1`
//
//	err := r.db.Get(&ticket, query, id)
//	if err != nil {
//		return nil, err
//	}
//	return &ticket, nil
//}
//func (r *TicketsRepo) Update(ticket *Ticket) error {
//	query := `
//        UPDATE tickets
//        SET price_amount = $2,
//            price_currency = $3,
//            customer_email = $4
//        WHERE ticket_id = $1`
//
//	_, err := r.db.Exec(query,
//		ticket.ID,
//		ticket.PriceAmount,
//		ticket.PriceCurrency,
//		ticket.CustomerEmail,
//	)
//	return err
//}
//
//func (r *TicketsRepo) Delete(id uuid.UUID) error {
//	query := `DELETE FROM tickets WHERE ticket_id = $1`
//	_, err := r.db.Exec(query, id)
//	return err
//}
//
//func (r *TicketsRepo) List() ([]Ticket, error) {
//	var tickets []Ticket
//	query := `
//        SELECT ticket_id, price_amount, price_currency, customer_email
//        FROM tickets`
//
//	err := r.db.Select(&tickets, query)
//	if err != nil {
//		return nil, err
//	}
//	return tickets, nil
//}
