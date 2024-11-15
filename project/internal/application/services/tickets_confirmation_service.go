package services

import (
	"context"
	"github.com/google/uuid"
	domain "tickets/internal/domain/tickets"
	"time"
)

type TicketConfirmationService struct {
	publisher domain.TicketBookingPublisher
}

func NewTicketConfirmationService(
	publisher domain.TicketBookingPublisher,
) *TicketConfirmationService {
	return &TicketConfirmationService{
		publisher: publisher,
	}
}

func (s *TicketConfirmationService) ConfirmTickets(ctx context.Context, tickets []domain.Ticket) {
	for _, ticket := range tickets {
		if ticket.Status == "confirmed" {
			s.publisher.PublishConfirmed(ctx, domain.TicketBookingConfirmedEvent{
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
			s.publisher.PublishCanceled(ctx, domain.TicketBookingCanceledEvent{
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
