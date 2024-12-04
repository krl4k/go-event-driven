package app

import (
	"context"
	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	_ "github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"os"
	"tickets/internal/application/services"
	"tickets/internal/infrastructure/event_publisher"
	"tickets/internal/interfaces/events"
	"tickets/internal/interfaces/http"
	"tickets/internal/repository"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type App struct {
	watermillLogger watermill.LoggerAdapter
	logger          zerolog.Logger
	router          *message.Router
	srv             *http.Server
	db              *sqlx.DB
}

func NewApp(
	watermillLogger watermill.LoggerAdapter,
	spreadsheetsClient events.SpreadsheetsService,
	receiptsClient events.ReceiptsService,
	filesClient events.FileStorageService,
	redisClient *redis.Client,
	db *sqlx.DB,
) (*App, error) {
	ticketsRepo := repository.NewTicketsRepo(db)
	showsRepo := repository.NewShowsRepo(db)
	bookingsRepo := repository.NewBookingsRepo(db)

	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return nil, err
	}

	var publisher message.Publisher
	publisher, err = redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: redisClient,
	}, watermillLogger)
	if err != nil {
		return nil, err
	}

	publisher = event_publisher.CorrelationPublisherDecorator{
		Publisher: publisher,
	}
	eventBus, err := NewEventBus(publisher, watermillLogger)

	ticketsService := services.NewTicketConfirmationService(eventBus, ticketsRepo)
	showsService := services.NewShowsService(showsRepo)
	bookingsService := services.NewBookingService(bookingsRepo)

	e := commonHTTP.NewEcho()
	srv := http.NewServer(
		e,
		ticketsService,
		showsService,
		bookingsService,
		router.IsRunning,
	)

	router.AddMiddleware(middleware.Recoverer)
	router.AddMiddleware(events.CorrelationIDMiddleware)
	router.AddMiddleware(events.LoggingMiddleware)

	router.AddMiddleware(middleware.Retry{
		MaxRetries:      10,
		InitialInterval: time.Millisecond * 100,
		MaxInterval:     time.Second,
		Multiplier:      2,
		Logger:          watermillLogger,
	}.Middleware)

	// skip marshalling errors before retrying
	router.AddMiddleware(events.SkipMarshallingErrorsMiddleware)

	marshaler := cqrs.JSONMarshaler{
		GenerateName: cqrs.StructName,
	}
	processor, err := NewEventProcessor(router, redisClient, marshaler, watermillLogger)
	processor.AddHandlers(
		events.TicketsToPrintHandler(spreadsheetsClient),
		events.PrepareTicketsHandler(filesClient, eventBus),
		events.IssueReceiptHandler(receiptsClient),
		events.StoreTicketsHandler(ticketsRepo),

		events.RefundTicketHandler(spreadsheetsClient),
		events.RemoveTicketsHandler(ticketsRepo),
	)

	return &App{
		watermillLogger: watermillLogger,
		logger:          zerolog.New(os.Stdout),
		router:          router,
		srv:             srv,
		db:              db,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	err := repository.InitializeDBSchema(a.db)
	if err != nil {
		return err
	}

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
	err = g.Wait()
	if err != nil {
		panic(err)
	}
	return nil
}

func NewEventBus(
	pub message.Publisher,
	logger watermill.LoggerAdapter,
) (*cqrs.EventBus, error) {
	return cqrs.NewEventBusWithConfig(
		pub,
		cqrs.EventBusConfig{
			GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
				return params.EventName, nil
			},
			Marshaler: cqrs.JSONMarshaler{
				GenerateName: cqrs.StructName,
			},
			Logger: logger,
		},
	)
}

func NewEventProcessor(
	router *message.Router,
	rdb *redis.Client,
	marshaler cqrs.CommandEventMarshaler,
	logger watermill.LoggerAdapter,
) (*cqrs.EventProcessor, error) {
	return cqrs.NewEventProcessorWithConfig(
		router,
		cqrs.EventProcessorConfig{
			GenerateSubscribeTopic: func(params cqrs.EventProcessorGenerateSubscribeTopicParams) (string, error) {
				return params.EventName, nil
			},
			SubscriberConstructor: func(params cqrs.EventProcessorSubscriberConstructorParams) (message.Subscriber, error) {
				return redisstream.NewSubscriber(redisstream.SubscriberConfig{
					Client:        rdb,
					ConsumerGroup: "svc-tickets." + params.HandlerName,
				}, logger)
			},
			Marshaler: marshaler,
			Logger:    logger,
		},
	)
}
