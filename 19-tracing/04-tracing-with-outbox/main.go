package main

import (
	"database/sql"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

var outboxTopic = "events_to_forward"

type PublisherWithTracing struct {
	message.Publisher
}

func (p *PublisherWithTracing) Publish(topic string, messages ...*message.Message) error {
	for i := range messages {
		otel.GetTextMapPropagator().
			Inject(messages[i].Context(), propagation.MapCarrier(messages[i].Metadata))

	}
	return p.Publisher.Publish(topic, messages...)
}

func PublishInTx(
	msg *message.Message,
	tx *sql.Tx,
	logger watermill.LoggerAdapter,
) error {
	var publisher message.Publisher
	var err error

	// this publisher publishes "enveloped" message,

	publisher, err = watermillSQL.NewPublisher(
		tx,
		watermillSQL.PublisherConfig{
			SchemaAdapter: watermillSQL.DefaultPostgreSQLSchema{},
		},
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create outbox publisher: %w", err)
	}

	// it's worth decorating as well to see forwarding operation in trace
	// (forwarding is done based on enveloped message)

	// this publisher publishes "our" message
	publisher = forwarder.NewPublisher(publisher, forwarder.PublisherConfig{
		ForwarderTopic: outboxTopic,
	})

	publisher = &PublisherWithTracing{publisher}

	// you should decorate your publisher here

	return publisher.Publish("ItemAddedToCart", msg)
}

func RunForwarder(
	db *sqlx.DB,
	rdb *redis.Client,
	outboxTopic string,
	logger watermill.LoggerAdapter,
) error {
	postgresSub, err := watermillSQL.NewSubscriber(
		db,
		watermillSQL.SubscriberConfig{
			SchemaAdapter:  watermillSQL.DefaultPostgreSQLSchema{},
			OffsetsAdapter: watermillSQL.DefaultPostgreSQLOffsetsAdapter{},
		},
		logger,
	)
	if err != nil {
		return err
	}

	err = postgresSub.SubscribeInitialize(outboxTopic)
	if err != nil {
		return err
	}

	redisPub, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: rdb,
	}, logger)
	if err != nil {
		return err
	}

	fwd, err := forwarder.NewForwarder(
		postgresSub,
		redisPub,
		logger,
		forwarder.Config{
			ForwarderTopic: outboxTopic,
			Middlewares: []message.HandlerMiddleware{
				func(h message.HandlerFunc) message.HandlerFunc {
					return func(msg *message.Message) ([]*message.Message, error) {
						fmt.Println("Forwarding message", msg.UUID, string(msg.Payload), msg.Metadata)

						return h(msg)
					}
				},
			},
		},
	)
	if err != nil {
		return err
	}

	go func() {
		err := fwd.Run(context.Background())
		if err != nil {
			panic(err)
		}
	}()

	<-fwd.Running()

	return nil
}
