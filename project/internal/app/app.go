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
	"time"
)

type App struct {
	watermillLogger watermill.LoggerAdapter
	logger          zerolog.Logger
	router          *message.Router
	srv             *http.Server
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

func NewApp(
	watermillLogger watermill.LoggerAdapter,
	spreadsheetsClient events.SpreadsheetsService,
	receiptsClient events.ReceiptsService,
	redisClient *redis.Client,
) (*App, error) {
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

	ticketConfirmationService := services.NewTicketConfirmationService(eventBus)
	e := commonHTTP.NewEcho()
	srv := http.NewServer(e, ticketConfirmationService, router.IsRunning)

	//appendToTrackerSubscriber, err := createSubscriber(redisClient, watermillLogger, "append-to-tracker-consumer-group")
	//if err != nil {
	//	return nil, err
	//}
	//
	//issueReceiptSubscriber, err := createSubscriber(redisClient, watermillLogger, "issue-receipt-consumer-group")
	//if err != nil {
	//	return nil, err
	//}

	router.AddMiddleware(middleware.Recoverer)
	router.AddMiddleware(events.CorrelationIDMiddleware)
	router.AddMiddleware(events.LoggingMiddleware)
	//router.AddMiddleware(MetadataTypeChecker)

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
		events.RefundTicketHandler(spreadsheetsClient),
		events.IssueReceiptHandler(receiptsClient),
	)

	//_ = events.NewEventHandlers(
	//	watermillLogger,
	//	router,
	//	appendToTrackerSubscriber,
	//	issueReceiptSubscriber,
	//	spreadsheetsClient,
	//	receiptsClient,
	//)

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
