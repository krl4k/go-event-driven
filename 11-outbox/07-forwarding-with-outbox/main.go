package main

import (
	"context"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"

	_ "github.com/lib/pq"
)

func RunForwarder(
	db *sqlx.DB,
	rdb *redis.Client,
	outboxTopic string,
	logger watermill.LoggerAdapter,
) error {
	subscriber, err := watermillSQL.NewSubscriber(
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

	err = subscriber.SubscribeInitialize(outboxTopic)
	if err != nil {
		return err
	}

	publisher, err := NewRedisPublisher(logger, rdb)
	if err != nil {
		return err
	}

	forwarder, err := forwarder.NewForwarder(subscriber, publisher,
		logger,
		forwarder.Config{
			ForwarderTopic: outboxTopic,
		},
	)
	if err != nil {
		return err
	}

	go func() {
		err := forwarder.Run(context.Background())
		if err != nil {
			panic(err)
		}
	}()

	<-forwarder.Running()
	return nil
}

func NewRedisPublisher(
	wlogger watermill.LoggerAdapter,
	redisClient *redis.Client,
) (message.Publisher, error) {
	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: redisClient,
	}, wlogger)
	if err != nil {
		return nil, err
	}

	return publisher, err
}
