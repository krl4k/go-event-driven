package main

import (
	"context"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
	"os"
	"tickets/internal/application/services"
	"tickets/internal/infrastructure/clients"
	"tickets/internal/infrastructure/event_publisher"
	"tickets/internal/interfaces/http"

	commonClients "github.com/ThreeDotsLabs/go-event-driven/common/clients"
	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"

	"github.com/sirupsen/logrus"
)

func main() {
	//logger := zerolog.New(os.Stdout)
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

	router, err := message.NewRouter(message.RouterConfig{}, wlogger)
	if err != nil {
		panic(err)
	}

	router.AddNoPublisherHandler(
		"append_to_tracker",
		"append-to-tracker",
		appendToTrackerSub,
		func(msg *message.Message) error {
			return spreadsheetsClient.AppendRow(
				msg.Context(),
				"tickets-to-print",
				[]string{string(msg.Payload)})
		},
	)

	router.AddNoPublisherHandler(
		"issue_receipt",
		"issue-receipt",
		issueReceiptSubscriber,
		func(msg *message.Message) error {
			return receiptsClient.IssueReceipt(
				msg.Context(),
				string(msg.Payload))
		},
	)

	go func() {
		if err := router.Run(context.Background()); err != nil {
			panic(err)
		}
	}()

	logrus.Info("Server starting...")
	srv.Start()
	// todo graceful shutdown
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
