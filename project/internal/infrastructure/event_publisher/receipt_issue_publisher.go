package event_publisher

import (
	"context"
	"encoding/json"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	domain "tickets/internal/domain/tickets"
)

type TicketBookingConfirmedPublisher struct {
	publisher message.Publisher
}

func NewTicketBookingConfirmedPublisher(publisher message.Publisher) *TicketBookingConfirmedPublisher {
	return &TicketBookingConfirmedPublisher{
		publisher: publisher,
	}
}

func (p *TicketBookingConfirmedPublisher) PublishConfirmed(ctx context.Context, event domain.TicketBookingConfirmedEvent) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := message.NewMessage(uuid.NewString(), bytes)

	msg.Metadata.Set("correlation_id", log.CorrelationIDFromContext(ctx))
	return p.publisher.Publish("TicketBookingConfirmed", msg)
}

func (p *TicketBookingConfirmedPublisher) PublishCanceled(ctx context.Context, event domain.TicketBookingCanceledEvent) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := message.NewMessage(uuid.NewString(), bytes)
	msg.Metadata.Set("correlation_id", log.CorrelationIDFromContext(ctx))

	return p.publisher.Publish("TicketBookingCanceled", msg)
}
