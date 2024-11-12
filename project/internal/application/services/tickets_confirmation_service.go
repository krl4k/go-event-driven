package services

import (
	"github.com/google/uuid"
	domain "tickets/internal/domain/tickets"
	"time"
)

type TicketConfirmationService struct {
	publisher domain.TicketBookingConfirmedPublisher
}

func NewTicketConfirmationService(
	publisher domain.TicketBookingConfirmedPublisher,
) *TicketConfirmationService {
	return &TicketConfirmationService{
		publisher: publisher,
	}
}

func (s *TicketConfirmationService) ConfirmTickets(tickets []domain.Ticket) {
	for _, ticket := range tickets {
		s.publisher.Publish(domain.TicketBookingConfirmedEvent{
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
