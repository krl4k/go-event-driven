package event_publisher

import (
	"encoding/json"
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

func (p *ReceiptIssuePublisher) PublishIssueReceipt(event domain.IssueReceiptEvent) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.publisher.Publish("issue-receipt", message.NewMessage(uuid.NewString(), bytes))
}
