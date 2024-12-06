package app

import (
	"context"
	"fmt"
	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	_ "github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"os"
	"tickets/internal/application/usecases/booking"
	"tickets/internal/application/usecases/shows"
	"tickets/internal/application/usecases/tickets"
	"tickets/internal/infrastructure/event_publisher"
	"tickets/internal/interfaces/events"
	"tickets/internal/interfaces/http"
	"tickets/internal/outbox"
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
	forwarder       *outbox.Forwarder
}

func NewApp(
	watermillLogger watermill.LoggerAdapter,
	spreadsheetsClient events.SpreadsheetsService,
	receiptsClient events.ReceiptsService,
	filesClient events.FileStorageService,
	deadNationClient events.DeadNationService,
	redisClient *redis.Client,
	db *sqlx.DB,
) (*App, error) {
	ticketsRepo := repository.NewTicketsRepo(db)
	showsRepo := repository.NewShowsRepo(db, trmsqlx.DefaultCtxGetter)
	bookingsRepo := repository.NewBookingsRepo(db, trmsqlx.DefaultCtxGetter)

	trManager := manager.Must(trmsqlx.NewDefaultFactory(db))

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
	eventBus, err := events.NewEventBus(publisher, watermillLogger)

	ticketsService := tickets.NewTicketConfirmationService(eventBus, ticketsRepo)
	showsService := shows.NewShowsService(showsRepo)
	bookingsService := booking.NewBookTicketsUsecase(
		bookingsRepo,
		showsRepo,
		trManager,
		trmsqlx.DefaultCtxGetter,
		watermillLogger,
	)

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
	processor, err := events.NewEventProcessor(router, redisClient, marshaler, watermillLogger)

	eventHandler := events.NewHandler(
		eventBus,
		spreadsheetsClient,
		receiptsClient,
		filesClient,
		deadNationClient,
		ticketsRepo,
		showsRepo,
	)

	processor.AddHandlers(
		// TicketBookingConfirmed handlers
		eventHandler.TicketsToPrintHandler(),
		eventHandler.PrepareTicketsHandler(),
		eventHandler.IssueReceiptHandler(),
		eventHandler.StoreTicketsHandler(),

		// TicketBookingCancelled handlers
		eventHandler.RefundTicketHandler(),
		eventHandler.RemoveTicketsHandler(),

		// BookingMade handlers
		eventHandler.TicketBookingHandler(),
	)

	forwarder, err := outbox.NewForwarder(
		db,
		redisClient,
		watermillLogger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create forwarder: %w", err)
	}

	return &App{
		watermillLogger: watermillLogger,
		logger:          zerolog.New(os.Stdout),
		router:          router,
		srv:             srv,
		db:              db,
		forwarder:       forwarder,
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
		a.logger.Info().Msg("starting outbox forwarder")
		a.forwarder.RunForwarder(ctx)

		a.logger.Info().Msg("forwarder is running")
		return nil
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
