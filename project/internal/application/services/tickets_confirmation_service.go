package services

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"
	domain "tickets/internal/domain/tickets"
	"time"
)

type TicketService struct {
	eb *cqrs.EventBus
}

func NewTicketConfirmationService(
	eb *cqrs.EventBus,
) *TicketService {
	return &TicketService{
		eb: eb,
	}
}

func (s *TicketService) ProcessTickets(ctx context.Context, tickets []domain.Ticket) {
	for _, ticket := range tickets {
		if ticket.Status == "confirmed" {
			s.eb.Publish(ctx, domain.TicketBookingConfirmed{
				Header: domain.Header{
					Id:          uuid.NewString(),
					PublishedAt: time.Now().Format(time.RFC3339),
				},
				TicketId:      ticket.TicketId,
				CustomerEmail: ticket.CustomerEmail,
				Price: domain.Money{
					Amount:   ticket.Price.Amount,
					Currency: ticket.Price.Currency,
				},
			})
		} else {
			s.eb.Publish(ctx, domain.TicketBookingCanceled{
				Header: domain.Header{
					Id:          uuid.NewString(),
					PublishedAt: time.Now().Format(time.RFC3339),
				},
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
