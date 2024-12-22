package outbox

import (
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"tickets/internal/observability"
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

	publisherWithTracing := observability.PublisherWithTracing{Publisher: publisher}

	fpublisher := forwarder.NewPublisher(publisherWithTracing, forwarder.PublisherConfig{
		ForwarderTopic: Topic,
	})

	return fpublisher, nil
}
