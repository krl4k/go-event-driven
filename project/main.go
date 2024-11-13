package main

import (
	"context"
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"os"
	"os/signal"
	"tickets/internal/application/services"
	domain "tickets/internal/domain/tickets"
	"tickets/internal/infrastructure/clients"
	"tickets/internal/infrastructure/event_publisher"
	"tickets/internal/interfaces/http"

	commonClients "github.com/ThreeDotsLabs/go-event-driven/common/clients"
	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
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

	tpublisher := event_publisher.NewTicketBookingConfirmedPublisher(publisher)

	ticketConfirmationService := services.NewTicketConfirmationService(tpublisher)

	router, err := message.NewRouter(message.RouterConfig{}, wlogger)
	if err != nil {
		panic(err)
	}

	router.AddMiddleware(func(next message.HandlerFunc) message.HandlerFunc {
		return func(message *message.Message) ([]*message.Message, error) {
			//logger.Info().
			//	Str("message_uuid", message.UUID).
			//	Msg("Handling a message")

			logrus.WithField("message_uuid", message.UUID).
				Info("Handling a message")
			return next(message)
		}
	})
	e := commonHTTP.NewEcho()
	srv := http.NewServer(e, ticketConfirmationService, router.IsRunning)

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

	router.AddNoPublisherHandler(
		"append_to_tracker",
		"TicketBookingConfirmed",
		appendToTrackerSub,
		func(msg *message.Message) error {
			var payload domain.TicketBookingConfirmedEvent
			err := json.Unmarshal(msg.Payload, &payload)
			if err != nil {
				return err
			}

			return spreadsheetsClient.AppendRow(
				msg.Context(),
				"tickets-to-print",
				[]string{
					payload.TicketId,
					payload.CustomerEmail,
					payload.Price.Amount,
					payload.Price.Currency})

		},
	)

	router.AddNoPublisherHandler(
		"refund_ticket",
		"TicketBookingCanceled",
		appendToTrackerSub,
		func(msg *message.Message) error {
			var payload domain.TicketBookingConfirmedEvent
			err := json.Unmarshal(msg.Payload, &payload)
			if err != nil {
				return err
			}

			return spreadsheetsClient.AppendRow(
				msg.Context(),
				"tickets-to-refund",
				[]string{
					payload.TicketId,
					payload.CustomerEmail,
					payload.Price.Amount,
					payload.Price.Currency})

		},
	)

	router.AddNoPublisherHandler(
		"issue_receipt",
		"TicketBookingConfirmed",
		issueReceiptSubscriber,
		func(msg *message.Message) error {
			var payload domain.TicketBookingConfirmedEvent
			err := json.Unmarshal(msg.Payload, &payload)
			if err != nil {
				return err
			}

			return receiptsClient.IssueReceipt(
				msg.Context(),
				clients.IssueReceiptRequest{
					TicketID: payload.TicketId,
					Price:    payload.Price,
				})
		},
	)

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		logger.Info().Msg("starting router")

		return router.Run(ctx)
	})

	g.Go(func() error {
		<-router.Running()
		logger.Info().Msg("router is running")

		logger.Info().Msg("starting server")
		return srv.Start()
	})

	g.Go(func() error {
		// Shut down
		<-ctx.Done()

		err = srv.Stop(ctx)
		if err != nil {
			logger.Err(err).Msg("error stopping server")
		}

		return err
	})

	// Will block until all goroutines finish
	err = g.Wait()
	if err != nil {
		panic(err)
	}
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
