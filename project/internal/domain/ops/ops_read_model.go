package ops

import (
	"github.com/google/uuid"
	"time"
)

type Booking struct {
	BookingID uuid.UUID `json:"booking_id"`
	BookedAt  time.Time `json:"booked_at"`

	Tickets map[string]Ticket `json:"tickets"`

	LastUpdate time.Time `json:"last_update"`
}

type Ticket struct {
	PriceAmount   string `json:"price_amount"`
	PriceCurrency string `json:"price_currency"`
	CustomerEmail string `json:"customer_email"`

	// Status should be set to "confirmed" or "refunded"
	Status string `json:"status"`

	PrintedAt       time.Time `json:"printed_at"`
	PrintedFileName string    `json:"printed_file_name"`

	ReceiptIssuedAt time.Time `json:"receipt_issued_at"`
	ReceiptNumber   string    `json:"receipt_number"`
}
