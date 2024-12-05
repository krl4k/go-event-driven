package events

import (
	"errors"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func MetadataTypeChecker(next message.HandlerFunc) message.HandlerFunc {
	return func(message *message.Message) ([]*message.Message, error) {
		if message.Metadata.Get("type") == "" {
			log.FromContext(message.Context()).
				Warn("Message type not set")

			return nil, nil
		}

		return next(message)
	}
}

func CorrelationIDMiddleware(next message.HandlerFunc) message.HandlerFunc {
	return func(message *message.Message) ([]*message.Message, error) {
		correlationID := message.Metadata.Get("correlation_id")

		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		ctx := log.ContextWithCorrelationID(message.Context(), correlationID)

		ctx = log.ToContext(ctx,
			logrus.WithFields(logrus.Fields{
				"correlation_id": correlationID,
				"message_uuid":   message.UUID,
			},
			))

		message.SetContext(ctx)

		return next(message)
	}
}

func LoggingMiddleware(next message.HandlerFunc) message.HandlerFunc {
	return func(message *message.Message) ([]*message.Message, error) {
		log.FromContext(message.Context()).
			WithField("payload", string(message.Payload)).
			WithField("metadata", message.Metadata).
			Info("Handling a message")

		messages, err := next(message)

		if err != nil {
			log.FromContext(message.Context()).
				WithField("payload", string(message.Payload)).
				WithField("error", err).
				Error("Message handling error")
		}

		return messages, err
	}
}

var ErrJsonUnmarshal = errors.New("json unmarshal error")

func SkipMarshallingErrorsMiddleware(h message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		msgs, err := h(msg)

		if err != nil {
			if errors.Is(err, ErrJsonUnmarshal) {
				log.FromContext(msg.Context()).
					WithField("error", err).
					Warn("Error while unmarshalling message")
				// skip this malformed message
				return nil, nil
			}
		}

		return msgs, err
	}
}
