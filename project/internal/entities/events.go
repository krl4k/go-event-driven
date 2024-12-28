package entities

import (
	"github.com/google/uuid"
	"time"
)

type Event interface {
	IsInternal() bool
}

type TicketBookingConfirmed_v1 struct {
	Header        EventHeader `json:"header"`
	TicketID      string      `json:"ticket_id"`
	CustomerEmail string      `json:"customer_email"`
	Price         Money       `json:"price"`
	BookingID     string      `json:"booking_id"`
}

func (t TicketBookingConfirmed_v1) IsInternal() bool {
	return false
}

type TicketBookingCanceled_v1 struct {
	Header        EventHeader `json:"header"`
	TicketId      string      `json:"ticket_id"`
	BookingId     string      `json:"booking_id"`
	CustomerEmail string      `json:"customer_email"`
	Price         Money       `json:"price"`
}

func (t TicketBookingCanceled_v1) IsInternal() bool {
	return false
}

type TicketPrinted_v1 struct {
	Header EventHeader `json:"header"`

	TicketID  string    `json:"ticket_id"`
	BookingID string    `json:"booking_id"`
	FileName  string    `json:"file_name"`
	PrintedAt time.Time `json:"printed_at"`
}

func (t TicketPrinted_v1) IsInternal() bool {
	return false
}

type TicketReceiptIssued_v1 struct {
	Header EventHeader `json:"header"`

	TicketId      string `json:"ticket_id"`
	ReceiptNumber string `json:"receipt_number"`

	IssuedAt  time.Time `json:"issued_at"`
	BookingId string    `json:"booking_id"`
}

func (t TicketReceiptIssued_v1) IsInternal() bool {
	return false
}

type TicketRefunded_v1 struct {
	Header EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
}

func (t TicketRefunded_v1) IsInternal() bool {
	return false
}

type BookingMade_v1 struct {
	Header          EventHeader `json:"header"`
	BookingID       uuid.UUID   `json:"booking_id"`
	NumberOfTickets int         `json:"number_of_tickets"`
	CustomerEmail   string      `json:"customer_email"`
	ShowID          uuid.UUID   `json:"show_id"`
	BookedAt        time.Time   `json:"booked_at"`
}

func (b BookingMade_v1) IsInternal() bool {
	return false
}

// events for data migration from datalake

type TicketBookingConfirmed_v0 struct {
	Header        EventHeader `json:"header"`
	TicketId      string      `json:"ticket_id"`
	CustomerEmail string      `json:"customer_email"`
	Price         Money       `json:"price"`
	BookingId     string      `json:"booking_id"`
}

func (t TicketBookingConfirmed_v0) IsInternal() bool {
	return false
}

type TicketBookingCanceled_v0 struct {
	Header        EventHeader `json:"header"`
	TicketId      string      `json:"ticket_id"`
	BookingId     string      `json:"booking_id"`
	CustomerEmail string      `json:"customer_email"`
	Price         Money       `json:"price"`
}

func (t TicketBookingCanceled_v0) IsInternal() bool {
	return false
}

type TicketPrinted_v0 struct {
	Header EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
	FileName string `json:"file_name"`
}

func (t TicketPrinted_v0) IsInternal() bool {
	return false
}

type TicketReceiptIssued_v0 struct {
	Header EventHeader `json:"header"`

	TicketId      string `json:"ticket_id"`
	ReceiptNumber string `json:"receipt_number"`

	IssuedAt time.Time `json:"issued_at"`
}

func (t TicketReceiptIssued_v0) IsInternal() bool {
	return false
}

type TicketRefunded_v0 struct {
	Header EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
}

func (t TicketRefunded_v0) IsInternal() bool {
	return false
}

type BookingMade_v0 struct {
	Header          EventHeader `json:"header"`
	BookingID       uuid.UUID   `json:"booking_id"`
	NumberOfTickets int         `json:"number_of_tickets"`
	CustomerEmail   string      `json:"customer_email"`
	ShowID          uuid.UUID   `json:"show_id"`
}

func (b BookingMade_v0) IsInternal() bool {
	return false
}
