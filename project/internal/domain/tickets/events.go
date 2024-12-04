package domain

import (
	"github.com/google/uuid"
	"time"
)

type EventHeader struct {
	Id             string `json:"id"`
	PublishedAt    string `json:"published_at"`
	IdempotencyKey string `json:"idempotency_key"`
}

func NewEventHeader() EventHeader {
	return EventHeader{
		Id:             uuid.NewString(),
		PublishedAt:    time.Now().Format(time.RFC3339),
		IdempotencyKey: uuid.NewString(),
	}
}

func NewEventHeaderWithIdempotencyKey(idempotencyKey string) EventHeader {
	return EventHeader{
		Id:             uuid.NewString(),
		PublishedAt:    time.Now().Format(time.RFC3339),
		IdempotencyKey: idempotencyKey,
	}
}

// event
type TicketBookingConfirmed struct {
	Header        EventHeader `json:"header"`
	TicketId      string      `json:"ticket_id"`
	CustomerEmail string      `json:"customer_email"`
	Price         Money       `json:"price"`
}

type TicketBookingCanceled struct {
	Header        EventHeader `json:"header"`
	TicketId      string      `json:"ticket_id"`
	CustomerEmail string      `json:"customer_email"`
	Price         Money       `json:"price"`
}

type TicketPrinted struct {
	Header EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
	FileName string `json:"file_name"`
}
