package domain

import "context"

type Header struct {
	Id          string `json:"id"`
	PublishedAt string `json:"published_at"`
}

type TicketBookingConfirmedEvent struct {
	Header        Header `json:"header"`
	TicketId      string `json:"ticket_id"`
	CustomerEmail string `json:"customer_email"`
	Price         Money  `json:"price"`
}

type TicketBookingCanceledEvent struct {
	Header        Header `json:"header"`
	TicketId      string `json:"ticket_id"`
	CustomerEmail string `json:"customer_email"`
	Price         Money  `json:"price"`
}

// Interfaces for domain events

type TicketBookingPublisher interface {
	PublishConfirmed(ctx context.Context, event TicketBookingConfirmedEvent) error
	PublishCanceled(ctx context.Context, event TicketBookingCanceledEvent) error
}
