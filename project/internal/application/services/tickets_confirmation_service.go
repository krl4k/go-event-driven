package services

import domain "tickets/internal/domain/tickets"

type TicketConfirmationService struct {
	receiptIssuePublisher    domain.ReceiptIssuePublisher
	appendToTrackerPublisher domain.AppendToTrackerPublisher
}

func NewTicketConfirmationService(
	receiptIssuePublisher domain.ReceiptIssuePublisher,
	appendToTrackerPublisher domain.AppendToTrackerPublisher,
) *TicketConfirmationService {
	return &TicketConfirmationService{
		receiptIssuePublisher:    receiptIssuePublisher,
		appendToTrackerPublisher: appendToTrackerPublisher,
	}
}

func (s *TicketConfirmationService) ConfirmTickets(tickets []domain.Ticket) {
	for _, ticket := range tickets {

		s.receiptIssuePublisher.PublishIssueReceipt(domain.IssueReceiptEvent{
			TicketId: ticket.TicketId,
			Price:    ticket.Price,
		})
		s.appendToTrackerPublisher.PublishAppendToTracker(domain.AppendToTrackerEvent{
			TicketId:      ticket.TicketId,
			CustomerEmail: ticket.CustomerEmail,
			Price:         ticket.Price,
		})
	}
}
