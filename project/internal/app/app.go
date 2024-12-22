package app

import (
	"context"
	"encoding/json"
	"fmt"
	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	_ "github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	watermillMessage "github.com/ThreeDotsLabs/watermill/message"
	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/sync/errgroup"
	"os"
	"tickets/internal/application/usecases/booking"
	"tickets/internal/application/usecases/shows"
	"tickets/internal/application/usecases/tickets"
	"tickets/internal/entities"
	"tickets/internal/infrastructure/event_publisher"
	"tickets/internal/interfaces/http"
	"tickets/internal/interfaces/message"
	"tickets/internal/interfaces/message/commands"
	events "tickets/internal/interfaces/message/events"
	outbox "tickets/internal/interfaces/message/outbox"
	"tickets/internal/observability"
	"tickets/internal/repository"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var (
	veryImportantCounter = promauto.NewCounter(prometheus.CounterOpts{
		// metric will be named tickets_very_important_counter_total
		Namespace: "tickets",
		Name:      "very_important_counter_total",
		Help:      "Total number of very important things processed",
	})
)

type App struct {
	watermillLogger         watermill.LoggerAdapter
	logger                  zerolog.Logger
	router                  *watermillMessage.Router
	srv                     *http.Server
	db                      *sqlx.DB
	eventsRepo              *repository.EventsRepository
	opsBookingReadModelRepo *repository.OpsBookingReadModelRepo
	traceProviver           *trace.TracerProvider
}

func NewApp(
	watermillLogger watermill.LoggerAdapter,
	spreadsheetsClient SpreadsheetsService,
	receiptsClient ReceiptsService,
	filesClient FileStorageService,
	deadNationClient DeadNationService,
	paymentsClient PaymentsService,
	redisClient *redis.Client,
	db *sqlx.DB,
	tp *trace.TracerProvider,
) (*App, error) {
	trManager := manager.Must(trmsqlx.NewDefaultFactory(db))
	var redisPublisher watermillMessage.Publisher
	redisPublisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: redisClient,
	}, watermillLogger)
	if err != nil {
		return nil, err
	}

	redisPublisher = event_publisher.CorrelationPublisherDecorator{
		Publisher: redisPublisher,
	}
	redisPublisher = observability.PublisherWithTracing{
		Publisher: redisPublisher,
	}
	eventBus, err := events.NewEventBus(redisPublisher, watermillLogger)

	ticketsRepo := repository.NewTicketsRepo(db)
	showsRepo := repository.NewShowsRepo(db, trmsqlx.DefaultCtxGetter)
	bookingsRepo := repository.NewBookingsRepo(db, trmsqlx.DefaultCtxGetter)
	opsBookingReadModelRepo := repository.NewOpsBookingReadModelRepo(
		db, trmsqlx.DefaultCtxGetter, trManager, eventBus)
	eventsRepo := repository.NewEventsRepo(db)

	commandBus, err := commands.NewBus(redisPublisher, watermillLogger)

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
		commandBus,
		ticketsService,
		showsService,
		bookingsService,
		opsBookingReadModelRepo,
	)

	redisSubscriber, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{Client: redisClient}, watermillLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber: %w", err)
	}

	postgresSubscriber, err := watermillSQL.NewSubscriber(
		db,
		watermillSQL.SubscriberConfig{
			SchemaAdapter:  watermillSQL.DefaultPostgreSQLSchema{},
			OffsetsAdapter: watermillSQL.DefaultPostgreSQLOffsetsAdapter{},
			// todo setup through config. for tests should be different values
			PollInterval:   500 * time.Millisecond,
			ResendInterval: 500 * time.Millisecond,
			RetryInterval:  500 * time.Millisecond,
		},
		watermillLogger,
	)
	if err != nil {
		return nil, err
	}

	err = postgresSubscriber.SubscribeInitialize(outbox.Topic)
	if err != nil {
		return nil, err
	}

	eventHandler := events.NewHandler(
		eventBus,
		spreadsheetsClient,
		receiptsClient,
		filesClient,
		deadNationClient,
		ticketsRepo,
		showsRepo,
	)

	commandHandler := commands.NewHandler(
		eventBus,
		paymentsClient,
		receiptsClient,
	)

	router, err := message.NewRouter(
		watermillLogger,
		postgresSubscriber,
		redisSubscriber,
		redisPublisher,

		eventHandler,
		commandHandler,

		cqrs.JSONMarshaler{
			GenerateName: cqrs.StructName,
		},
		events.NewEventProcessorConfig(redisClient, watermillLogger),
		commands.NewCommandProcessorConfig(redisClient, watermillLogger),
		eventsRepo,
		opsBookingReadModelRepo,
	)
	if err != nil {
		return nil, err
	}

	return &App{
		watermillLogger:         watermillLogger,
		logger:                  zerolog.New(os.Stdout),
		router:                  router,
		srv:                     srv,
		db:                      db,
		eventsRepo:              eventsRepo,
		opsBookingReadModelRepo: opsBookingReadModelRepo,
		traceProviver:           tp,
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

		err := a.router.Run(ctx)
		if err != nil {
			a.logger.Err(err).Msg("error running router")
		}
		return nil
	})

	g.Go(func() error {
		<-a.router.Running()
		a.logger.Info().Msg("router is running")

		a.logger.Info().Msg("starting server")
		return a.srv.Start()
	})

	g.Go(func() error {
		a.logger.Info().Msg("migrating events")
		err := a.MigrateEvents()
		if err != nil {
			a.logger.Err(err).Msg("failed to migrate events")
		}
		return err
	})

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				veryImportantCounter.Inc()
			}
			time.Sleep(time.Millisecond * 100)
		}
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

	g.Go(func() error {
		<-ctx.Done()
		err := a.traceProviver.Shutdown(ctx)
		if err != nil {
			return err
		}
		return nil
	})

	// Will block until all goroutines finish
	err = g.Wait()
	if err != nil {
		a.logger.Err(err).Msg("error running app")
		return err
	}
	return nil
}

func (a *App) MigrateEvents() error {
	// if table not empty, migrate events
	// if empty -> wait for events

	for {
		isEmpty, err := a.eventsRepo.IsEmpty(context.Background())
		if err != nil {
			return fmt.Errorf("failed to check if events table is empty: %w", err)
		}

		if !isEmpty {
			break
		}

		log.FromContext(context.Background()).Info("Waiting for events")

		time.Sleep(100 * time.Millisecond)

		// get all events, type switch and save in the ops read model
		log.FromContext(context.Background()).Info("Getting events")
		events, err := a.eventsRepo.GetEvents(context.Background())
		if err != nil {
			log.FromContext(context.Background()).Info("Failed to get events: ", err)
			return err
		}

		log.FromContext(context.Background()).Info("Migrating events, count: ", len(events))

		for _, event := range events {
			log.FromContext(context.Background()).Info("Processing event: ", event.EventName, " with payload: ", string(event.Payload))
			switch event.EventName {
			case "BookingMade_v0":
				var bookingMade entities.BookingMade_v0
				err := json.Unmarshal(event.Payload, &bookingMade)
				if err != nil {
					return fmt.Errorf("failed to unmarshal BookingMade_v0 event: %w", err)
				}

				err = a.opsBookingReadModelRepo.OnBookingMadeV0Event(context.Background(), &bookingMade)
				if err != nil {
					return fmt.Errorf("failed to handle BookingMade_v0 event: %w", err)
				}
			case "TicketBookingConfirmed_v0":
				var ticketBookingConfirmed entities.TicketBookingConfirmed_v0
				err := json.Unmarshal(event.Payload, &ticketBookingConfirmed)
				if err != nil {
					return fmt.Errorf("failed to unmarshal TicketBookingConfirmed_v0 event: %w", err)
				}

				err = a.opsBookingReadModelRepo.OnTicketBookingConfirmedV0Event(context.Background(), &ticketBookingConfirmed)
				if err != nil {
					return fmt.Errorf("failed to handle TicketBookingConfirmed_v0 event: %w", err)
				}

			case "TicketReceiptIssued_v0":
				var ticketReceiptIssued entities.TicketReceiptIssued_v0
				err := json.Unmarshal(event.Payload, &ticketReceiptIssued)
				if err != nil {
					return fmt.Errorf("failed to unmarshal TicketReceiptIssued_v0 event: %w", err)
				}

				err = a.opsBookingReadModelRepo.OnTicketReceiptIssuedV0Event(context.Background(), &ticketReceiptIssued)
				if err != nil {
					return fmt.Errorf("failed to handle TicketReceiptIssued_v0 event: %w", err)
				}
			case "TicketPrinted_v0":
				var ticketPrinted entities.TicketPrinted_v0
				err := json.Unmarshal(event.Payload, &ticketPrinted)
				if err != nil {
					return fmt.Errorf("failed to unmarshal TicketPrinted_v0 event: %w", err)
				}

				err = a.opsBookingReadModelRepo.OnTicketPrintedV0Event(context.Background(), &ticketPrinted)
				if err != nil {
					return fmt.Errorf("failed to handle TicketPrinted_v0 event: %w", err)
				}
			case "TicketRefunded_v0":
				var ticketBookingCanceled entities.TicketRefunded_v0
				err := json.Unmarshal(event.Payload, &ticketBookingCanceled)
				if err != nil {
					return fmt.Errorf("failed to unmarshal TicketRefunded_v0 event: %w", err)
				}

				err = a.opsBookingReadModelRepo.OnTicketRefundedV0Event(context.Background(), &ticketBookingCanceled)
				if err != nil {
					return fmt.Errorf("failed to handle TicketRefunded_v0 event: %w", err)
				}
			}
		}
	}

	return nil
}
