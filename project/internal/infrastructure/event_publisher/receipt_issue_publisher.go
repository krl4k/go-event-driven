package event_publisher

import (
	"encoding/json"
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

func (p *TicketBookingConfirmedPublisher) Publish(event domain.TicketBookingConfirmedEvent) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.publisher.Publish("TicketBookingConfirmed", message.NewMessage(uuid.NewString(), bytes))
}
