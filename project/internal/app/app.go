package app

import (
	"context"
	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	_ "github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"os"
	"tickets/internal/application/services"
	"tickets/internal/infrastructure/event_publisher"
	"tickets/internal/interfaces/events"
	"tickets/internal/interfaces/http"
)

type App struct {
	watermillLogger watermill.LoggerAdapter
	logger          zerolog.Logger
	router          *message.Router
	srv             *http.Server
}

func NewApp(
	watermillLogger watermill.LoggerAdapter,
	spreadsheetsClient events.SpreadsheetsAPI,
	receiptsClient events.ReceiptsService,
	redisClient *redis.Client,
) (*App, error) {
	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return nil, err
	}

	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: redisClient,
	}, watermillLogger)
	if err != nil {
		return nil, err
	}

	tpublisher := event_publisher.NewTicketBookingConfirmedPublisher(publisher)
	_ = services.NewTicketConfirmationService(tpublisher)

	ticketConfirmationService := services.NewTicketConfirmationService(tpublisher)
	e := commonHTTP.NewEcho()
	srv := http.NewServer(e, ticketConfirmationService, router.IsRunning)

	appendToTrackerSubscriber, err := createSubscriber(redisClient, watermillLogger, "append-to-tracker-consumer-group")
	if err != nil {
		return nil, err
	}

	issueReceiptSubscriber, err := createSubscriber(redisClient, watermillLogger, "issue-receipt-consumer-group")
	if err != nil {
		return nil, err
	}

	_ = events.NewEventHandlers(
		watermillLogger,
		router,
		appendToTrackerSubscriber,
		issueReceiptSubscriber,
		spreadsheetsClient,
		receiptsClient,
	)

	return &App{
		watermillLogger: watermillLogger,
		logger:          zerolog.New(os.Stdout),
		router:          router,
		srv:             srv,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		a.logger.Info().Msg("starting router")

		return a.router.Run(ctx)
	})

	g.Go(func() error {
		<-a.router.Running()
		a.logger.Info().Msg("router is running")

		a.logger.Info().Msg("starting server")
		return a.srv.Start()
	})

	g.Go(func() error {
		// Shut down
		<-ctx.Done()

		err := a.srv.Stop(ctx)
		if err != nil {
			a.logger.Err(err).Msg("error stopping server")
		}

		return err
	})

	// Will block until all goroutines finish
	err := g.Wait()
	if err != nil {
		panic(err)
	}
	return nil
}

func createSubscriber(
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
