package bookings

import "github.com/google/uuid"

type Booking struct {
	Id              uuid.UUID `json:"id"`
	ShowId          uuid.UUID `json:"show_id"`
	NumberOfTickets int       `json:"number_of_tickets"`
	CustomerEmail   string    `json:"customer_email"`
}
