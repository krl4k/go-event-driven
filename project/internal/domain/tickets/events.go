package domain

import (
	domain2 "tickets/internal/domain"
	"time"
)

type TicketBookingConfirmed struct {
	Header        domain2.EventHeader `json:"header"`
	TicketId      string              `json:"ticket_id"`
	CustomerEmail string              `json:"customer_email"`
	Price         Money               `json:"price"`
	BookingId     string              `json:"booking_id"`
}

type TicketBookingCanceled struct {
	Header        domain2.EventHeader `json:"header"`
	TicketId      string              `json:"ticket_id"`
	BookingId     string              `json:"booking_id"`
	CustomerEmail string              `json:"customer_email"`
	Price         Money               `json:"price"`
}

type TicketPrinted struct {
	Header domain2.EventHeader `json:"header"`

	TicketID  string    `json:"ticket_id"`
	BookingID string    `json:"booking_id"`
	FileName  string    `json:"file_name"`
	PrintedAt time.Time `json:"printed_at"`
}

type TicketReceiptIssued struct {
	Header domain2.EventHeader `json:"header"`

	TicketId      string `json:"ticket_id"`
	ReceiptNumber string `json:"receipt_number"`

	IssuedAt  time.Time `json:"issued_at"`
	BookingId string    `json:"booking_id"`
}

type TicketRefunded struct {
	Header domain2.EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
}
