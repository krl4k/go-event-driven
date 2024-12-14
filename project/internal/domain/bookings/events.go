package bookings

import (
	"github.com/google/uuid"
	"tickets/internal/domain"
	"time"
)

type BookingMade_v1 struct {
	Header          domain.EventHeader `json:"header"`
	BookingID       uuid.UUID          `json:"booking_id"`
	NumberOfTickets int                `json:"number_of_tickets"`
	CustomerEmail   string             `json:"customer_email"`
	ShowID          uuid.UUID          `json:"show_id"`
	BookedAt        time.Time          `json:"booked_at"`
}
