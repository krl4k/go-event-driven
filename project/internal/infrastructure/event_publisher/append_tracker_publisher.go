package event_publisher

import (
	"encoding/json"
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

func (p *ReceiptIssuePublisher) PublishAppendToTracker(event domain.AppendToTrackerEvent) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.publisher.Publish("append-to-tracker", message.NewMessage(uuid.NewString(), bytes))
}
