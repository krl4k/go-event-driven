package tickets

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"tickets/internal/entities"
	"tickets/internal/idempotency"
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
