package message

import (
	"fmt"
	"tickets/internal/entities"
	"tickets/internal/interfaces/message/commands"
	"tickets/internal/interfaces/message/events"
	"tickets/internal/interfaces/message/outbox"
	"tickets/internal/repository"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func NewRouter(
	watermillLogger watermill.LoggerAdapter,
	postgresSubscriber message.Subscriber,
	redisSubscriber message.Subscriber,
	redisPublisher message.Publisher,

	eventHandler *events.Handler,
	commandsHandler *commands.Handler,

	marshaller cqrs.CommandEventMarshaler,
	eventProcessorConfig cqrs.EventProcessorConfig,
	commandProcessorConfig cqrs.CommandProcessorConfig,

	eventsRepo events.EventRepository,
	opsBookingReadModelRepo *repository.OpsBookingReadModelRepo,
	vipBundleProcessManager *events.VipBundleProcessManager,
) (*message.Router, error) {

	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return nil, err
	}

	initMiddlewares(watermillLogger, router)

	outbox.AddForwarderHandler(
		postgresSubscriber,
		redisPublisher,
		router,
		watermillLogger,
	)

	eventProcessor, err := cqrs.NewEventProcessorWithConfig(router, eventProcessorConfig)
	if err != nil {
		return nil, err
	}

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

		// VIP bundle handlers
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.on_vip_bundle_initialized",
			vipBundleProcessManager.OnVipBundleInitialized,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.on_booking_made",
			vipBundleProcessManager.OnBookingMade,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.on_ticket_booking_confirmed",
			vipBundleProcessManager.OnTicketBookingConfirmed,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.on_flight_booked",
			vipBundleProcessManager.OnFlightBooked,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.on_taxi_booked",
			vipBundleProcessManager.OnTaxiBooked,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.on_booking_failed",
			vipBundleProcessManager.OnBookingFailed,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.on_flight_booking_failed",
			vipBundleProcessManager.OnFlightBookingFailed,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.on_taxi_booking_failed",
			vipBundleProcessManager.OnTaxiBookingFailed,
		),

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

	commandsProcessor, err := cqrs.NewCommandProcessorWithConfig(router, commandProcessorConfig)
	if err != nil {
		return nil, err
	}
	err = commandsProcessor.AddHandlers(
		commandsHandler.RefundTicketsHandler(),
		commandsHandler.BookShowTicketsHandler(),
		commandsHandler.BookFlightHandler(),
		commandsHandler.BookTaxiHandler(),
	)
	if err != nil {
		return nil, err
	}

	router.AddNoPublisherHandler(
		"events_splitter",
		"events",
		redisSubscriber,
		func(msg *message.Message) error {
			eventName := marshaller.NameFromMessage(msg)
			if eventName == "" {
				return fmt.Errorf("cannot get event name from message")
			}

			return redisPublisher.Publish("events."+eventName, msg)
		},
	)

	router.AddNoPublisherHandler(
		"events_saver",
		"events",
		redisSubscriber,
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

			id, err := uuid.Parse(event.Header.Id)
			if err != nil {
				return fmt.Errorf("failed to parse BookingID: %w", err)
			}

			err = eventsRepo.SaveEvent(
				msg.Context(),
				entities.DatalakeEvent{
					Id:          id,
					PublishedAt: event.Header.PublishedAt,
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

	return router, nil
}

func initMiddlewares(watermillLogger watermill.LoggerAdapter, router *message.Router) {
	router.AddMiddleware(events.TracingMiddleware)
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
	router.AddMiddleware(events.MetricsMiddleware)
}
