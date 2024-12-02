package event_publisher

import (
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/message"
)

type CorrelationPublisherDecorator struct {
	message.Publisher
}

func (c CorrelationPublisherDecorator) Publish(topic string, messages ...*message.Message) error {
	for _, msg := range messages {
		msg.Metadata.Set("correlation_id", log.CorrelationIDFromContext(msg.Context()))
	}
	return c.Publisher.Publish(topic, messages...)
}
