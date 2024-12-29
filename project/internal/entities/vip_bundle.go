package entities

import (
	"fmt"
	"github.com/google/uuid"
	"time"
)

type VipBundle struct {
	VipBundleID uuid.UUID `json:"vip_bundle_id"`

	BookingID       uuid.UUID  `json:"booking_id"`
	CustomerEmail   string     `json:"customer_email"`
	NumberOfTickets int        `json:"number_of_tickets"`
	ShowId          uuid.UUID  `json:"show_id"`
	BookingMadeAt   *time.Time `json:"booking_made_at"`

	TicketIDs []uuid.UUID `json:"ticket_ids"`

	Passengers []string `json:"passengers"`

	InboundFlightID         uuid.UUID   `json:"inbound_flight_id"`
	InboundFlightBookedAt   *time.Time  `json:"inbound_flight_booked_at"`
	InboundFlightTicketsIDs []uuid.UUID `json:"inbound_flight_tickets_ids"`

	ReturnFlightID         uuid.UUID   `json:"return_flight_id"`
	ReturnFlightBookedAt   *time.Time  `json:"return_flight_booked_at"`
	ReturnFlightTicketsIDs []uuid.UUID `json:"return_flight_tickets_ids"`

	TaxiBookedAt  *time.Time `json:"taxi_booked_at"`
	TaxiBookingID *uuid.UUID `json:"taxi_booking_id"`

	IsFinalized bool `json:"finalized"`
	Failed      bool `json:"failed"`
}

func NewVipBundle(
	vipBundleID uuid.UUID,
	bookingID uuid.UUID,
	customerEmail string,
	numberOfTickets int,
	showId uuid.UUID,
	passengers []string,
	inboundFlightID uuid.UUID,
	returnFlightID uuid.UUID,
) (*VipBundle, error) {
	if vipBundleID == uuid.Nil {
		return nil, fmt.Errorf("vip bundle id must be set")
	}
	if bookingID == uuid.Nil {
		return nil, fmt.Errorf("booking id must be set")
	}
	if customerEmail == "" {
		return nil, fmt.Errorf("customer email must be set")
	}
	if numberOfTickets <= 0 {
		return nil, fmt.Errorf("number of tickets must be greater than 0")
	}
	if showId == uuid.Nil {
		return nil, fmt.Errorf("show id must be set")
	}
	if numberOfTickets != len(passengers) {
		return nil, fmt.Errorf("number of tickets and passengers count mismatch")
	}
	if inboundFlightID == uuid.Nil {
		return nil, fmt.Errorf("inbound flight id must be set")
	}
	if returnFlightID == uuid.Nil {
		return nil, fmt.Errorf("return flight id must be set")
	}

	return &VipBundle{
		VipBundleID:     vipBundleID,
		BookingID:       bookingID,
		CustomerEmail:   customerEmail,
		NumberOfTickets: numberOfTickets,
		ShowId:          showId,
		Passengers:      passengers,
		InboundFlightID: inboundFlightID,
		ReturnFlightID:  returnFlightID,
	}, nil
}

type BookShowTickets struct {
	BookingID uuid.UUID `json:"booking_id"`

	CustomerEmail   string    `json:"customer_email"`
	NumberOfTickets int       `json:"number_of_tickets"`
	ShowId          uuid.UUID `json:"show_id"`
}

func (b BookShowTickets) IsInternal() bool {
	return false
}

type BookFlight struct {
	CustomerEmail  string    `json:"customer_email"`
	FlightID       uuid.UUID `json:"to_flight_id"`
	Passengers     []string  `json:"passengers"`
	ReferenceID    string    `json:"reference_id"`
	IdempotencyKey string    `json:"idempotency_key"`
}

type BookTaxi struct {
	CustomerEmail      string `json:"customer_email"`
	CustomerName       string `json:"customer_name"`
	NumberOfPassengers int    `json:"number_of_passengers"`
	ReferenceID        string `json:"reference_id"`
	IdempotencyKey     string `json:"idempotency_key"`
}

type CancelFlightTickets struct {
	FlightTicketIDs []uuid.UUID `json:"flight_ticket_id"`
}

type VipBundleInitialized_v1 struct {
	Header EventHeader `json:"header"`

	VipBundleID uuid.UUID `json:"vip_bundle_id"`
}

func (t VipBundleInitialized_v1) IsInternal() bool {
	return false
}

type BookingFailed_v1 struct {
	Header EventHeader `json:"header"`

	BookingID     uuid.UUID `json:"booking_id"`
	FailureReason string    `json:"failure_reason"`
}

type FlightBooked_v1 struct {
	Header EventHeader `json:"header"`

	FlightID  uuid.UUID   `json:"flight_id"`
	TicketIDs []uuid.UUID `json:"flight_tickets_ids"`

	ReferenceID string `json:"reference_id"`
}

func (t FlightBooked_v1) IsInternal() bool {
	return false
}

type FlightBookingFailed_v1 struct {
	Header EventHeader `json:"header"`

	FlightID      uuid.UUID `json:"flight_id"`
	FailureReason string    `json:"failure_reason"`

	// used for tieing events from our system with external systems
	// will be returned from the external system in the response and used
	// to find the corresponding entity
	ReferenceID string `json:"reference_id"`
}

func (t FlightBookingFailed_v1) IsInternal() bool {
	return false
}

type TaxiBooked_v1 struct {
	Header EventHeader `json:"header"`

	TaxiBookingID uuid.UUID `json:"taxi_booking_id"`

	ReferenceID string `json:"reference_id"`
}

func (t TaxiBooked_v1) IsInternal() bool {
	return false
}

type VipBundleFinalized_v1 struct {
	Header EventHeader `json:"header"`

	VipBundleID uuid.UUID `json:"vip_bundle_id"`
}

func (t VipBundleFinalized_v1) IsInternal() bool {
	return false
}

type TaxiBookingFailed_v1 struct {
	Header EventHeader `json:"header"`

	FailureReason string `json:"failure_reason"`

	ReferenceID string `json:"reference_id"`
}

func (t TaxiBookingFailed_v1) IsInternal() bool {
	return false
}
