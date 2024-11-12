package domain

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
	PublishConfirmed(event TicketBookingConfirmedEvent) error
	PublishCanceled(event TicketBookingCanceledEvent) error
}
