package bookings

import "github.com/google/uuid"

type BookingMade struct {
	BookingID       uuid.UUID `json:"booking_id"`
	NumberOfTickets int       `json:"number_of_tickets"`
	CustomerEmail   string    `json:"customer_email"`
	ShowID          uuid.UUID `json:"show_id"`
}
