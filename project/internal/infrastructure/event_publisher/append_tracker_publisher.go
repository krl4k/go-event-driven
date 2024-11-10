package event_publisher

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	domain "tickets/internal/domain/tickets"
)

type AppendTrackerPublisher struct {
	publisher message.Publisher
}

func NewAppendTrackerPublisher(publisher message.Publisher) *ReceiptIssuePublisher {
	return &ReceiptIssuePublisher{
		publisher: publisher,
	}
}

func (p *ReceiptIssuePublisher) PublishAppendToTracker(ticket domain.Ticket) error {
	return p.publisher.Publish("append-to-tracker", message.NewMessage(uuid.NewString(), []byte(ticket)))
}
