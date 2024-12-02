package domain

type Header struct {
	Id          string `json:"id"`
	PublishedAt string `json:"published_at"`
}

// event
type TicketBookingConfirmed struct {
	Header        Header `json:"header"`
	TicketId      string `json:"ticket_id"`
	CustomerEmail string `json:"customer_email"`
	Price         Money  `json:"price"`
}

type TicketBookingCanceled struct {
	Header        Header `json:"header"`
	TicketId      string `json:"ticket_id"`
	CustomerEmail string `json:"customer_email"`
	Price         Money  `json:"price"`
}
