package tickets

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	domain2 "tickets/internal/domain"
	domain "tickets/internal/domain/tickets"
	"tickets/internal/idempotency"
)

type TicketsRepository interface {
	List(ctx context.Context) ([]domain.Ticket, error)
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
	tickets []domain.Ticket,
) error {
	for _, ticket := range tickets {
		if ticket.Status == "confirmed" {
			err := s.eb.Publish(ctx, domain.TicketBookingConfirmed{
				Header: domain2.NewEventHeaderWithIdempotencyKey(
					idempotency.GetKey(ctx) + ticket.TicketId,
				),
				TicketId:      ticket.TicketId,
				CustomerEmail: ticket.CustomerEmail,
				Price: domain.Money{
					Amount:   ticket.Price.Amount,
					Currency: ticket.Price.Currency,
				},
				BookingId: ticket.BookingId,
			})
			if err != nil {
				return fmt.Errorf("failed to publish TicketBookingConfirmed: %w", err)
			}
		} else {
			err := s.eb.Publish(ctx, domain.TicketBookingCanceled{
				Header: domain2.NewEventHeaderWithIdempotencyKey(
					idempotency.GetKey(ctx) + ticket.TicketId,
				),
				TicketId:      ticket.TicketId,
				CustomerEmail: ticket.CustomerEmail,
				Price: domain.Money{
					Amount:   ticket.Price.Amount,
					Currency: ticket.Price.Currency,
				},
				BookingId: ticket.BookingId,
			})
			if err != nil {
				return fmt.Errorf("failed to publish TicketBookingCanceled: %w", err)
			}
		}
	}
	return nil
}

func (s *ProcessTicketsUsecase) GetTickets(ctx context.Context) ([]domain.Ticket, error) {
	tickets, err := s.ticketsRepo.List(ctx)

	return tickets, err
}
