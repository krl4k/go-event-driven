package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type EventHeader struct {
	ID             string    `json:"id"`
	PublishedAt    time.Time `json:"published_at"`
	IdempotencyKey string    `json:"idempotency_key"`
}

func NewEventHeader() EventHeader {
	return EventHeader{
		ID:             uuid.NewString(),
		PublishedAt:    time.Now(),
		IdempotencyKey: uuid.NewString(),
	}
}

type Money struct {
	Amount   string `json:"amount" db:"amount"`
	Currency string `json:"currency" db:"currency"`
}

type BookShowTickets struct {
	BookingID uuid.UUID `json:"booking_id"`

	CustomerEmail   string    `json:"customer_email"`
	NumberOfTickets int       `json:"number_of_tickets"`
	ShowId          uuid.UUID `json:"show_id"`
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

type RefundTicket struct {
	Header EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
}

type BookingMade_v1 struct {
	Header EventHeader `json:"header"`

	NumberOfTickets int `json:"number_of_tickets"`

	BookingID uuid.UUID `json:"booking_id"`

	CustomerEmail string    `json:"customer_email"`
	ShowId        uuid.UUID `json:"show_id"`
}

type TicketBookingConfirmed_v1 struct {
	Header EventHeader `json:"header"`

	TicketID      string `json:"ticket_id"`
	CustomerEmail string `json:"customer_email"`
	Price         Money  `json:"price"`

	BookingID string `json:"booking_id"`
}

type VipBundleInitialized_v1 struct {
	Header EventHeader `json:"header"`

	VipBundleID uuid.UUID `json:"vip_bundle_id"`
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

type FlightBookingFailed_v1 struct {
	Header EventHeader `json:"header"`

	FlightID      uuid.UUID `json:"flight_id"`
	FailureReason string    `json:"failure_reason"`

	ReferenceID string `json:"reference_id"`
}

type TaxiBooked_v1 struct {
	Header EventHeader `json:"header"`

	TaxiBookingID uuid.UUID `json:"taxi_booking_id"`

	ReferenceID string `json:"reference_id"`
}

type VipBundleFinalized_v1 struct {
	Header EventHeader `json:"header"`

	VipBundleID uuid.UUID `json:"vip_bundle_id"`
}

type TaxiBookingFailed_v1 struct {
	Header EventHeader `json:"header"`

	FailureReason string `json:"failure_reason"`

	ReferenceID string `json:"reference_id"`
}

type CommandBus interface {
	Send(ctx context.Context, command any) error
}

type EventBus interface {
	Publish(ctx context.Context, event any) error
}

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
		return nil, fmt.Errorf("")
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

type VipBundleRepository interface {
	Add(ctx context.Context, vipBundle VipBundle) error
	Get(ctx context.Context, vipBundleID uuid.UUID) (VipBundle, error)
	GetByBookingID(ctx context.Context, bookingID uuid.UUID) (VipBundle, error)

	UpdateByID(
		ctx context.Context,
		bookingID uuid.UUID,
		updateFn func(vipBundle VipBundle) (VipBundle, error),
	) (VipBundle, error)

	UpdateByBookingID(
		ctx context.Context,
		bookingID uuid.UUID,
		updateFn func(vipBundle VipBundle) (VipBundle, error),
	) (VipBundle, error)
}

type VipBundleProcessManager struct {
	commandBus CommandBus
	eventBus   EventBus
	repository VipBundleRepository
}

func NewVipBundleProcessManager(
	commandBus CommandBus,
	eventBus EventBus,
	repository VipBundleRepository,
) *VipBundleProcessManager {
	return &VipBundleProcessManager{
		commandBus: commandBus,
		eventBus:   eventBus,
		repository: repository,
	}
}

func (v VipBundleProcessManager) OnVipBundleInitialized(ctx context.Context, event *VipBundleInitialized_v1) error {
	vpBundle, err := v.repository.Get(ctx, event.VipBundleID)
	if err != nil {
		return fmt.Errorf("OnVipBundleInitialized: get vip bundle: %w", err)
	}

	err = v.commandBus.Send(ctx, BookShowTickets{
		BookingID:       vpBundle.BookingID,
		CustomerEmail:   vpBundle.CustomerEmail,
		NumberOfTickets: vpBundle.NumberOfTickets,
		ShowId:          vpBundle.ShowId,
	})
	if err != nil {
		return fmt.Errorf("OnVipBundleInitialized: sending book show tickets: %w", err)
	}

	return err
}

func (v VipBundleProcessManager) OnBookingMade(ctx context.Context, event *BookingMade_v1) error {
	vpBundle, err := v.repository.UpdateByBookingID(ctx, event.BookingID, func(vipBundle VipBundle) (VipBundle, error) {
		vipBundle.BookingMadeAt = &event.Header.PublishedAt
		return vipBundle, nil
	})
	if err != nil {
		return fmt.Errorf("OnBookingMade: update vip bundle: %w", err)
	}

	// book inbound flight
	err = v.commandBus.Send(ctx, BookFlight{
		CustomerEmail:  vpBundle.CustomerEmail,
		FlightID:       vpBundle.InboundFlightID,
		Passengers:     vpBundle.Passengers,
		ReferenceID:    vpBundle.VipBundleID.String(),
		IdempotencyKey: event.Header.IdempotencyKey,
	})
	if err != nil {
		return fmt.Errorf("OnBookingMade: sending book flight: %w", err)
	}

	return nil
}

func (v VipBundleProcessManager) OnTicketBookingConfirmed(ctx context.Context, event *TicketBookingConfirmed_v1) error {
	bookingID, err := uuid.Parse(event.BookingID)
	if err != nil {
		return fmt.Errorf("OnTicketBookingConfirmed: parse booking id: %w", err)
	}

	_, err = v.repository.UpdateByBookingID(ctx, bookingID, func(vipBundle VipBundle) (VipBundle, error) {
		vipBundle.TicketIDs = append(vipBundle.TicketIDs, uuid.MustParse(event.TicketID))
		return vipBundle, nil
	})
	if err != nil {
		return fmt.Errorf("OnTicketBookingConfirmed: update vip bundle: %w", err)
	}

	return nil
}

func (v VipBundleProcessManager) OnFlightBooked(ctx context.Context, event *FlightBooked_v1) error {
	vpBundlerID, err := uuid.Parse(event.ReferenceID)
	if err != nil {
		return fmt.Errorf("OnFlightBooked: parse reference id: %w", err)
	}
	vpBundle, err := v.repository.Get(ctx, vpBundlerID)
	if err != nil {
		return fmt.Errorf("OnFlightBooked: get vip bundle: %w", err)
	}

	switch event.FlightID {
	case vpBundle.InboundFlightID:
		vpBundle.InboundFlightBookedAt = &event.Header.PublishedAt
		vpBundle.InboundFlightTicketsIDs = event.TicketIDs

		_, err = v.repository.UpdateByID(ctx, vpBundlerID, func(vipBundle VipBundle) (VipBundle, error) {
			return vpBundle, nil
		})
		if err != nil {
			return fmt.Errorf("OnFlightBooked: update vip bundle: %w", err)
		}

		// book return flight
		err = v.commandBus.Send(ctx, BookFlight{
			CustomerEmail:  vpBundle.CustomerEmail,
			FlightID:       vpBundle.ReturnFlightID,
			Passengers:     vpBundle.Passengers,
			ReferenceID:    vpBundle.VipBundleID.String(),
			IdempotencyKey: event.Header.IdempotencyKey,
		})
		if err != nil {
			return fmt.Errorf("OnBookingMade: sending book flight: %w", err)
		}
		return nil
	case vpBundle.ReturnFlightID:
		vpBundle.ReturnFlightBookedAt = &event.Header.PublishedAt
		vpBundle.ReturnFlightTicketsIDs = event.TicketIDs
		_, err = v.repository.UpdateByID(ctx, vpBundlerID, func(vipBundle VipBundle) (VipBundle, error) {
			return vpBundle, nil
		})
		if err != nil {
			return fmt.Errorf("OnFlightBooked: update vip bundle: %w", err)
		}
	default:
		return fmt.Errorf("OnFlightBooked: unknown flight id: %s", event.FlightID.String())
	}

	if vpBundle.InboundFlightBookedAt != nil && vpBundle.ReturnFlightBookedAt != nil {
		err = v.commandBus.Send(ctx, BookTaxi{
			CustomerEmail:      vpBundle.CustomerEmail,
			CustomerName:       vpBundle.Passengers[0],
			NumberOfPassengers: vpBundle.NumberOfTickets,
			ReferenceID:        vpBundle.VipBundleID.String(),
			IdempotencyKey:     event.Header.IdempotencyKey,
		})
		if err != nil {
			return fmt.Errorf("OnFlightBooked: sending book taxi: %w", err)
		}
	}

	return nil
}

func (v VipBundleProcessManager) OnTaxiBooked(ctx context.Context, event *TaxiBooked_v1) error {
	vpBundleID, err := uuid.Parse(event.ReferenceID)
	if err != nil {
		return fmt.Errorf("OnTaxiBooked: parse reference id: %w", err)
	}
	_, err = v.repository.UpdateByID(ctx, vpBundleID, func(vipBundle VipBundle) (VipBundle, error) {
		vipBundle.TaxiBookedAt = &event.Header.PublishedAt
		vipBundle.TaxiBookingID = &event.TaxiBookingID

		vipBundle.IsFinalized = true
		return vipBundle, nil
	})
	if err != nil {
		return fmt.Errorf("OnTaxiBooked: update vip bundle: %w", err)
	}

	err = v.eventBus.Publish(ctx, VipBundleFinalized_v1{
		Header:      NewEventHeader(),
		VipBundleID: vpBundleID,
	})
	if err != nil {
		return fmt.Errorf("OnTaxiBooked: sending vip bundle finalized: %w", err)
	}

	return nil
}

func (v VipBundleProcessManager) OnBookingFailed(ctx context.Context, event *BookingFailed_v1) error {
	vpBundle, err := v.repository.GetByBookingID(ctx, event.BookingID)
	if err != nil {
		return fmt.Errorf("OnBookingFailed: get vip bundle: %w", err)
	}

	err = v.rollback(ctx, vpBundle)
	if err != nil {
		return fmt.Errorf("OnBookingFailed: rollback: %w", err)
	}

	return nil
}

func (v VipBundleProcessManager) OnFlightBookingFailed(ctx context.Context, event *FlightBookingFailed_v1) error {
	vpBundleID, err := uuid.Parse(event.ReferenceID)
	if err != nil {
		return fmt.Errorf("OnFlightBookingFailed: parse reference id: %w", err)
	}
	vpBundle, err := v.repository.Get(ctx, vpBundleID)
	if err != nil {
		return fmt.Errorf("OnFlightBookingFailed: get vip bundle: %w", err)
	}

	err = v.rollback(ctx, vpBundle)
	if err != nil {
		return fmt.Errorf("OnFlightBookingFailed: rollback: %w", err)
	}
	return nil
}

func (v VipBundleProcessManager) OnTaxiBookingFailed(ctx context.Context, event *TaxiBookingFailed_v1) error {
	vpBundleID, err := uuid.Parse(event.ReferenceID)
	if err != nil {
		return fmt.Errorf("OnTaxiBookingFailed: parse reference id: %w", err)
	}
	vpBundle, err := v.repository.Get(ctx, vpBundleID)
	if err != nil {
		return fmt.Errorf("OnTaxiBookingFailed: get vip bundle: %w", err)
	}

	err = v.rollback(ctx, vpBundle)
	if err != nil {
		return fmt.Errorf("OnTaxiBookingFailed: rollback: %w", err)
	}
	return nil
}

func (v VipBundleProcessManager) rollback(ctx context.Context, vpBundle VipBundle) error {
	if vpBundle.BookingMadeAt != nil &&
		(vpBundle.TicketIDs == nil || len(vpBundle.TicketIDs) != vpBundle.NumberOfTickets) {
		return fmt.Errorf("rollback: number of tickets and ticket ids mismatch")
	}

	for _, ticketID := range vpBundle.TicketIDs {
		err := v.commandBus.Send(ctx, RefundTicket{
			Header:   NewEventHeader(),
			TicketID: ticketID.String(),
		})
		if err != nil {
			return fmt.Errorf("rollback: sending refund ticket: %w", err)
		}
	}

	if vpBundle.InboundFlightTicketsIDs != nil {
		// rollback inbound flight
		err := v.commandBus.Send(ctx, CancelFlightTickets{
			FlightTicketIDs: vpBundle.InboundFlightTicketsIDs,
		})
		if err != nil {
			return fmt.Errorf("rollback: sending cancel inbound flight tickets: %w", err)
		}
	}

	if vpBundle.ReturnFlightTicketsIDs != nil {
		// rollback return flight
		err := v.commandBus.Send(ctx, CancelFlightTickets{
			FlightTicketIDs: vpBundle.ReturnFlightTicketsIDs,
		})
		if err != nil {
			return fmt.Errorf("rollback: sending cancel return flight tickets: %w", err)
		}
	}

	_, err := v.repository.UpdateByBookingID(ctx, vpBundle.BookingID, func(vipBundle VipBundle) (VipBundle, error) {
		vipBundle.Failed = true
		vipBundle.IsFinalized = true
		return vipBundle, nil
	})
	if err != nil {
		return fmt.Errorf("rollback: update vip bundle: %w", err)
	}

	return nil
}
