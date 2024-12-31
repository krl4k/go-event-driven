package clients

import (
	"context"
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/transportation"
	"github.com/google/uuid"
)

var ErrFlightAlreadyBooked = fmt.Errorf("flight tickets already booked")

type TransportationClient struct {
	clients *clients.Clients
}

func NewTransportationClient(clients *clients.Clients) TransportationClient {
	return TransportationClient{
		clients: clients,
	}
}

type BookFlightTicketRequest struct {
	CustomerEmail  string
	FlightID       uuid.UUID
	PassengerNames []string
	ReferenceId    string
	IdempotencyKey string
}

type BookFlightTicketResponse struct {
	TicketsID []uuid.UUID
}

func (c TransportationClient) BookFlightTicket(ctx context.Context, request *BookFlightTicketRequest) (*BookFlightTicketResponse, error) {
	resp, err := c.clients.Transportation.PutFlightTicketsWithResponse(ctx, transportation.BookFlightTicketRequest{
		CustomerEmail:  request.CustomerEmail,
		FlightId:       request.FlightID,
		PassengerNames: request.PassengerNames,
		ReferenceId:    request.ReferenceId,
		IdempotencyKey: request.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == 409 {
		return nil, ErrFlightAlreadyBooked
	}

	if resp.JSON201 == nil {
		return nil, fmt.Errorf("unexpected response: %v", resp)
	}
	return &BookFlightTicketResponse{
		TicketsID: resp.JSON201.TicketIds,
	}, nil
}

type BookTaxiRequest struct {
	CustomerEmail      string
	NumberOfPassengers int
	PassengerName      string
	ReferenceId        string
	IdempotencyKey     string
}

type BookTaxiResponse struct {
	BookingID uuid.UUID
}

var ErrTaxiAlreadyBooked = errors.New("taxi already booked")

func (c TransportationClient) BookTaxi(ctx context.Context, request *BookTaxiRequest) (*BookTaxiResponse, error) {
	resp, err := c.clients.Transportation.PutTaxiBookingWithResponse(ctx, transportation.TaxiBookingRequest{
		CustomerEmail:      request.CustomerEmail,
		NumberOfPassengers: request.NumberOfPassengers,
		PassengerName:      request.PassengerName, // this should be name of the first passenger in Vip Bundle
		ReferenceId:        request.ReferenceId,
		IdempotencyKey:     request.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == 409 {
		return nil, ErrTaxiAlreadyBooked
	}

	if resp.JSON201 == nil {
		return nil, fmt.Errorf("unexpected response: %v", resp)
	}

	return &BookTaxiResponse{
		BookingID: resp.JSON201.BookingId,
	}, nil
}

func (c TransportationClient) CancelFlightTickets(ctx context.Context, ticketID uuid.UUID) error {
	_, err := c.clients.Transportation.DeleteFlightTicketsTicketIdWithResponse(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to cancel flight tickets: %w", err)
	}

	if err != nil {
		return fmt.Errorf("delete flight: %w", err)
	}

	return nil
}
