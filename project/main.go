package main

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"os"
	"tickets/internal/application/services"
	"tickets/internal/infrastructure/clients"
	"tickets/internal/infrastructure/event_publisher"
	eventHandlers "tickets/internal/interfaces/events"
	"tickets/internal/interfaces/http"

	commonClients "github.com/ThreeDotsLabs/go-event-driven/common/clients"
	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"

	"github.com/sirupsen/logrus"
)

func main() {
	logger := zerolog.New(os.Stdout)
	wlogger := watermill.NewStdLogger(false, false)

	commonClients, err := commonClients.NewClients(os.Getenv("GATEWAY_ADDR"), nil)
	if err != nil {
		panic(err)
	}

	receiptsClient := clients.NewReceiptsClient(commonClients)
	spreadsheetsClient := clients.NewSpreadsheetsClient(commonClients)

	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: rdb,
	}, wlogger)

	appendTrackerPublisher := event_publisher.NewAppendTrackerPublisher(publisher)
	receiptIssuePublisher := event_publisher.NewReceiptIssuePublisher(publisher)

	ticketConfirmationService := services.NewTicketConfirmationService(receiptIssuePublisher, appendTrackerPublisher)

	e := commonHTTP.NewEcho()
	srv := http.NewServer(e, ticketConfirmationService)

	appendToTrackerSub, err := createSubscribers(rdb, wlogger, "append-to-tracker-consumer-group")
	if err != nil {
		wlogger.Error("error creating subscriber", err, nil)
		return
	}
	issueReceiptSubscriber, err := createSubscribers(rdb, wlogger, "issue-receipt-consumer-group")
	if err != nil {
		wlogger.Error("error creating subscriber", err, nil)
		return
	}

	appendToTrackerHandler, err := eventHandlers.NewAppendToTrackerHandler(logger, appendToTrackerSub, spreadsheetsClient)
	if err != nil {
		logger.Err(err).Msg("error creating append-to-tracker-handler")
		return
	}
	issueReceiptHandler, err := eventHandlers.NewIssueReceiptHandler(logger, issueReceiptSubscriber, receiptsClient)
	if err != nil {
		logger.Err(err).Msg("error creating issue-receipt-handler")
		return
	}

	go appendToTrackerHandler.Run()
	go issueReceiptHandler.Run()

	logrus.Info("Server starting...")
	srv.Start()
	// graceful shutdown

}

func createSubscribers(
	rdb *redis.Client,
	logger watermill.LoggerAdapter,
	consumerGroup string,
) (*redisstream.Subscriber, error) {
	sub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        rdb,
		ConsumerGroup: consumerGroup,
	}, logger)
	if err != nil {
		return nil, err
	}

	return sub, nil
}
