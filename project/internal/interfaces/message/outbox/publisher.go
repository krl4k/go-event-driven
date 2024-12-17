package outbox

import (
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
)

func NewPublisher(
	tx watermillSQL.ContextExecutor,
	logger watermill.LoggerAdapter,
) (*forwarder.Publisher, error) {
	publisher, err := watermillSQL.NewPublisher(
		tx,
		watermillSQL.PublisherConfig{
			SchemaAdapter: watermillSQL.DefaultPostgreSQLSchema{},
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	fpublisher := forwarder.NewPublisher(publisher, forwarder.PublisherConfig{
		ForwarderTopic: Topic,
	})

	return fpublisher, nil
}

//func PublishInTx(
//	topic string,
//	tx watermillSQL.ContextExecutor,
//	msg *message.Message,
//	logger watermill.LoggerAdapter,
//) error {
//
//	return fpublisher.Publish(topic, msg)
//}
