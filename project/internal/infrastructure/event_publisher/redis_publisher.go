package event_publisher

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
)

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
