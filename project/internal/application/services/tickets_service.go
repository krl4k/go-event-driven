package services

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	domain "tickets/internal/domain/tickets"
	"tickets/internal/idempotency"
)

type TicketsRepository interface {
	List(ctx context.Context) ([]domain.Ticket, error)
}

type TicketService struct {
	eb          *cqrs.EventBus
	ticketsRepo TicketsRepository
}

func NewTicketConfirmationService(
	eb *cqrs.EventBus,
	ticketsRepo TicketsRepository,
) *TicketService {
	return &TicketService{
		eb:          eb,
		ticketsRepo: ticketsRepo,
	}
}

func (s *TicketService) ProcessTickets(
	ctx context.Context,
	tickets []domain.Ticket,
) {
	for _, ticket := range tickets {
		if ticket.Status == "confirmed" {
			s.eb.Publish(ctx, domain.TicketBookingConfirmed{
				Header: domain.NewEventHeaderWithIdempotencyKey(
					idempotency.GetKey(ctx) + ticket.TicketId,
				),
				TicketId:      ticket.TicketId,
				CustomerEmail: ticket.CustomerEmail,
				Price: domain.Money{
					Amount:   ticket.Price.Amount,
					Currency: ticket.Price.Currency,
				},
			})
		} else {
			s.eb.Publish(ctx, domain.TicketBookingCanceled{
				Header: domain.NewEventHeaderWithIdempotencyKey(
					idempotency.GetKey(ctx) + ticket.TicketId,
				),
				TicketId:      ticket.TicketId,
				CustomerEmail: ticket.CustomerEmail,
				Price: domain.Money{
					Amount:   ticket.Price.Amount,
					Currency: ticket.Price.Currency,
				},
			})
		}
	}
}

func (s *TicketService) GetTickets(ctx context.Context) ([]domain.Ticket, error) {
	tickets, err := s.ticketsRepo.List(ctx)

	return tickets, err
}
