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

func (s *TicketConfirmationService) ConfirmTickets(tickets []string) {
	for _, ticketID := range tickets {
		ticket := domain.Ticket(ticketID)

		s.receiptIssuePublisher.PublishIssueReceipt(ticket)
		s.appendToTrackerPublisher.PublishAppendToTracker(ticket)
	}
}
