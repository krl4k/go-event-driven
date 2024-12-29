package tickets

import (
	"context"
	"fmt"
	"tickets/internal/entities"
	"tickets/internal/idempotency"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

type TicketsRepository interface {
	List(ctx context.Context) ([]entities.Ticket, error)
}

type ProcessTicketsUsecase struct {
	eb          *cqrs.EventBus
	ticketsRepo TicketsRepository
}

func NewTicketConfirmationService(
	eb *cqrs.EventBus,
	ticketsRepo TicketsRepository,
) *ProcessTicketsUsecase {
	return &ProcessTicketsUsecase{
		eb:          eb,
		ticketsRepo: ticketsRepo,
	}
}

func (s *ProcessTicketsUsecase) ProcessTickets(
	ctx context.Context,
	tickets []entities.Ticket,
) error {
	for _, ticket := range tickets {
		if ticket.Status == "confirmed" {
			log.FromContext(ctx).Info("Publishing TicketBookingConfirmed_v1 with id:"+ticket.TicketId, " Booking BookingID: ", ticket.BookingId)
			err := s.eb.Publish(ctx, entities.TicketBookingConfirmed_v1{
				Header: entities.NewEventHeaderWithIdempotencyKey(
					idempotency.GetKey(ctx) + ticket.TicketId,
				),
				TicketID:      ticket.TicketId,
				CustomerEmail: ticket.CustomerEmail,
				Price: entities.Money{
					Amount:   ticket.Price.Amount,
					Currency: ticket.Price.Currency,
				},
				BookingID: ticket.BookingId,
			})
			if err != nil {
				return fmt.Errorf("failed to publish TicketBookingConfirmed_v1: %w", err)
			}
		} else {
			err := s.eb.Publish(ctx, entities.TicketBookingCanceled_v1{
				Header: entities.NewEventHeaderWithIdempotencyKey(
					idempotency.GetKey(ctx) + ticket.TicketId,
				),
				TicketId:      ticket.TicketId,
				CustomerEmail: ticket.CustomerEmail,
				Price: entities.Money{
					Amount:   ticket.Price.Amount,
					Currency: ticket.Price.Currency,
				},
				BookingId: ticket.BookingId,
			})
			if err != nil {
				return fmt.Errorf("failed to publish TicketBookingCanceled_v1: %w", err)
			}
		}
	}
	return nil
}

func (s *ProcessTicketsUsecase) GetTickets(ctx context.Context) ([]entities.Ticket, error) {
	tickets, err := s.ticketsRepo.List(ctx)

	return tickets, err
}

type Service struct {
	transportationClient TransportationClient
	// add other dependencies as needed
}

func NewService(transportationClient TransportationClient) *Service {
	return &Service{
		transportationClient: transportationClient,
	}
}

type TransportationClient interface {
	BookTaxi(ctx context.Context, request BookTaxiRequest) error
	BookFlight(ctx context.Context, request BookFlightRequest) error
}

type BookTicketCommand struct {
	TicketID string
	// Add other fields you need
}

type BookFlightCommand struct {
	TicketID string
	// Add other fields you need
}

func (s *Service) BookTicket(ctx context.Context, cmd BookTicketCommand) error {
	// Implement taxi booking logic
	return s.transportationClient.BookTaxi(ctx, BookTaxiRequest{
		TicketID: cmd.TicketID,
		// Add other fields
	})
}

func (s *Service) BookFlight(ctx context.Context, cmd BookFlightCommand) error {
	// Implement flight booking logic
	return s.transportationClient.BookFlight(ctx, BookFlightRequest{
		TicketID: cmd.TicketID,
		// Add other fields
	})
}

type BookTaxiRequest struct {
	TicketID string
	// Add other fields
}

type BookFlightRequest struct {
	TicketID string
	// Add other fields
}
