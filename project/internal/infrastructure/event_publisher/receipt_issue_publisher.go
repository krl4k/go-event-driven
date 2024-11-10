package event_publisher

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	domain "tickets/internal/domain/tickets"
)

type ReceiptIssuePublisher struct {
	publisher message.Publisher
}

func NewReceiptIssuePublisher(publisher message.Publisher) *ReceiptIssuePublisher {
	return &ReceiptIssuePublisher{
		publisher: publisher,
	}
}

func (p *ReceiptIssuePublisher) PublishIssueReceipt(ticket domain.Ticket) error {
	return p.publisher.Publish("issue-receipt", message.NewMessage(uuid.NewString(), []byte(ticket)))
}
