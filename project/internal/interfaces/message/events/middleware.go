package events

import (
	"errors"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"time"
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
	return func(msg *message.Message) ([]*message.Message, error) {
		topic := message.SubscribeTopicFromCtx(msg.Context())
		handler := message.HandlerNameFromCtx(msg.Context())

		log.FromContext(msg.Context()).
			WithField("topic", topic).
			WithField("handler", handler).
			WithField("payload", string(msg.Payload)).
			WithField("metadata", msg.Metadata).
			Info("Handling a message")

		messages, err := next(msg)

		if err != nil {
			log.FromContext(msg.Context()).
				WithField("topic", topic).
				WithField("handler", handler).
				WithField("payload", string(msg.Payload)).
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

var (
	messagesProcessedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "messages_processed_total",
		Help: "Total number of messages processed",
	}, []string{"topic", "handler"})
	messagesProcessingFailedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "messages_processing_failed_total",
		Help: "Total number of messages processing failures",
	}, []string{"topic", "handler"})

	messagesProcessingDuration = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name:       "messages_processing_duration_seconds",
		Help:       "Duration of message processing in seconds",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"topic", "handler"})
)

func MetricsMiddleware(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		topic := message.SubscribeTopicFromCtx(msg.Context())
		handler := message.HandlerNameFromCtx(msg.Context())

		start := time.Now()

		msgs, err := next(msg)

		duration := time.Since(start)
		messagesProcessingDuration.WithLabelValues(topic, handler).Observe(duration.Seconds())

		messagesProcessedTotal.WithLabelValues(topic, handler).Inc()

		if err != nil {
			messagesProcessingFailedTotal.WithLabelValues(topic, handler).Inc()
		}

		return msgs, err
	}
}

func TracingMiddleware(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) (events []*message.Message, err error) {
		topic := message.SubscribeTopicFromCtx(msg.Context())
		handler := message.HandlerNameFromCtx(msg.Context())

		ctx := msg.Context()
		ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(msg.Metadata))

		ctx, span := otel.Tracer("").Start(
			ctx,
			fmt.Sprintf("topic: %s, handler: %s", topic, handler),
		)
		defer span.End()

		msg.SetContext(ctx)

		events, err = next(msg)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return events, err
	}
}
