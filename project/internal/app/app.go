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
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"os"
	"tickets/internal/application/usecases/booking"
	"tickets/internal/application/usecases/shows"
	"tickets/internal/application/usecases/tickets"
	"tickets/internal/entities"
	"tickets/internal/infrastructure/event_publisher"
	"tickets/internal/interfaces/commands"
	"tickets/internal/interfaces/events"
	"tickets/internal/interfaces/http"
	"tickets/internal/outbox"
	"tickets/internal/repository"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type App struct {
	watermillLogger         watermill.LoggerAdapter
	logger                  zerolog.Logger
	router                  *message.Router
	srv                     *http.Server
	db                      *sqlx.DB
	forwarder               *outbox.Forwarder
	eventsRepo              *repository.EventsRepository
	opsBookingReadModelRepo *repository.OpsBookingReadModelRepo
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
) (*App, error) {
	trManager := manager.Must(trmsqlx.NewDefaultFactory(db))
	var publisher message.Publisher
	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: redisClient,
	}, watermillLogger)
	if err != nil {
		return nil, err
	}

	publisher = event_publisher.CorrelationPublisherDecorator{
		Publisher: publisher,
	}
	eventBus, err := events.NewEventBus(publisher, watermillLogger)

	ticketsRepo := repository.NewTicketsRepo(db)
	showsRepo := repository.NewShowsRepo(db, trmsqlx.DefaultCtxGetter)
	bookingsRepo := repository.NewBookingsRepo(db, trmsqlx.DefaultCtxGetter)
	opsBookingReadModelRepo := repository.NewOpsBookingReadModelRepo(
		db, trmsqlx.DefaultCtxGetter, trManager, eventBus)
	eventsRepo := repository.NewEventsRepo(db)

	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return nil, err
	}

	commandBus, err := commands.NewBus(publisher, watermillLogger)

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

	sub, err :=
		redisstream.NewSubscriber(redisstream.SubscriberConfig{
			Client:        redisClient,
			ConsumerGroup: "events_splitter",
		}, watermillLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber: %w", err)
	}

	marshaller := cqrs.JSONMarshaler{
		GenerateName: cqrs.StructName,
	}

	router.AddNoPublisherHandler(
		"events_splitter",
		"events",
		sub,
		func(msg *message.Message) error {
			eventName := marshaller.NameFromMessage(msg)
			if eventName == "" {
				return fmt.Errorf("cannot get event name from message")
			}

			return publisher.Publish("events."+eventName, msg)
		},
	)

	saverSub, err :=
		redisstream.NewSubscriber(redisstream.SubscriberConfig{
			Client:        redisClient,
			ConsumerGroup: "events_saver",
		}, watermillLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber: %w", err)
	}

	router.AddNoPublisherHandler(
		"events_saver",
		"events",
		saverSub,
		func(msg *message.Message) error {
			type Event struct {
				Header entities.EventHeader `json:"header"`
			}

			var event Event
			err := marshaller.Unmarshal(msg, &event)
			if err != nil {
				return err
			}

			eventName := marshaller.NameFromMessage(msg)
			if eventName == "" {
				return fmt.Errorf("cannot get event name from message")
			}

			publishedAt, err := time.Parse(time.RFC3339, event.Header.PublishedAt)
			if err != nil {
				return fmt.Errorf("failed to parse published_at for event %s: %w", eventName, err)
			}

			id, err := uuid.Parse(event.Header.Id)
			if err != nil {
				return fmt.Errorf("failed to parse ID: %w", err)
			}

			err = eventsRepo.SaveEvent(
				msg.Context(),
				entities.DatalakeEvent{
					Id:          id,
					PublishedAt: publishedAt,
					EventName:   eventName,
					Payload:     msg.Payload,
				},
			)
			if err != nil {
				return fmt.Errorf("failed to save event %s: %w", eventName, err)
			}

			return err
		},
	)

	eventHandler := events.NewHandler(
		eventBus,
		spreadsheetsClient,
		receiptsClient,
		filesClient,
		deadNationClient,
		ticketsRepo,
		showsRepo,
	)

	eventProcessor, err := events.NewEventProcessor(router, redisClient, marshaller, watermillLogger)

	eventProcessor.AddHandlers(
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

		// Read model handlers
		cqrs.NewEventHandler(
			"ops_booking_read_model.on_booking_made",
			opsBookingReadModelRepo.OnBookingMadeEvent),
		cqrs.NewEventHandler(
			"ops_booking_read_model.on_ticket_booking_confirmed",
			opsBookingReadModelRepo.OnTicketBookingConfirmedEvent),
		cqrs.NewEventHandler(
			"ops_booking_read_model.on_ticket_receipt_issued",
			opsBookingReadModelRepo.OnTicketReceiptIssuedEvent),
		cqrs.NewEventHandler(
			"ops_booking_read_model.on_ticket_printed",
			opsBookingReadModelRepo.OnTicketPrintedEvent),
		cqrs.NewEventHandler(
			"ops_booking_read_model.on_ticket_removed",
			opsBookingReadModelRepo.OnTicketRefundedEvent),
	)

	commandHandlers := commands.NewHandler(eventBus, paymentsClient, receiptsClient)
	commandsProcessor, err := commands.NewCommandsProcessor(router, redisClient, watermillLogger)
	if err != nil {
		return nil, err
	}
	err = commandsProcessor.AddHandlers(
		commandHandlers.RefundTicketsHandler(),
	)
	if err != nil {
		return nil, err
	}

	forwarder, err := outbox.NewForwarder(
		db,
		redisClient,
		watermillLogger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create forwarder: %w", err)
	}

	return &App{
		watermillLogger:         watermillLogger,
		logger:                  zerolog.New(os.Stdout),
		router:                  router,
		srv:                     srv,
		db:                      db,
		forwarder:               forwarder,
		eventsRepo:              eventsRepo,
		opsBookingReadModelRepo: opsBookingReadModelRepo,
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
		a.logger.Info().Msg("migrating events")
		err := a.MigrateEvents()
		if err != nil {
			a.logger.Err(err).Msg("failed to migrate events")
		}
		return err
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
