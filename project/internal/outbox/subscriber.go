package outbox

import (
	"context"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"time"
)

type Forwarder struct {
	logger watermill.LoggerAdapter
	fwd    *forwarder.Forwarder
}

func NewForwarder(
	db *sqlx.DB,
	rdb *redis.Client,
	logger watermill.LoggerAdapter,
) (*Forwarder, error) {
	subscriber, err := watermillSQL.NewSubscriber(
		db,
		watermillSQL.SubscriberConfig{
			SchemaAdapter:  watermillSQL.DefaultPostgreSQLSchema{},
			OffsetsAdapter: watermillSQL.DefaultPostgreSQLOffsetsAdapter{},
			// todo setup through config. for tests should be different values
			PollInterval:   100 * time.Millisecond,
			ResendInterval: 100 * time.Millisecond,
			RetryInterval:  100 * time.Millisecond,
		},
		logger,
	)
	if err != nil {
		return nil, err
	}

	err = subscriber.SubscribeInitialize(outbox_topic)
	if err != nil {
		return nil, err
	}

	publisher, err := NewRedisPublisher(logger, rdb)
	if err != nil {
		return nil, err
	}

	forwarder, err := forwarder.NewForwarder(subscriber, publisher,
		logger,
		forwarder.Config{
			ForwarderTopic: outbox_topic,
		},
	)
	if err != nil {
		return nil, err
	}

	return &Forwarder{
		fwd:    forwarder,
		logger: logger,
	}, nil
}

func (f *Forwarder) RunForwarder(ctx context.Context) {
	go func() {
		err := f.fwd.Run(ctx)
		if err != nil {
			panic(err)
		}
	}()

	<-f.fwd.Running()
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
