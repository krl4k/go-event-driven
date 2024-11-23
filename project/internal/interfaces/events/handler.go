package events

import (
	"context"
	"encoding/json"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	domain "tickets/internal/domain/tickets"
	"time"
)

//go:generate mockgen -destination=mocks/spreadsheets_service_mock.go -package=mocks . SpreadsheetsService
type SpreadsheetsService interface {
	AppendRow(ctx context.Context, req domain.AppendToTrackerRequest) error
}

//go:generate mockgen -destination=mocks/receipts_service_mock.go -package=mocks . ReceiptsService
type ReceiptsService interface {
	IssueReceipt(ctx context.Context, request domain.IssueReceiptRequest) (*domain.IssueReceiptResponse, error)
}

type EventHandlers struct {
	router                 *message.Router
	appendToTrackerSub     message.Subscriber
	issueReceiptSubscriber message.Subscriber
	spreadsheetsClient     SpreadsheetsService
	receiptsClient         ReceiptsService
}

func NewEventHandlers(
	wlogger watermill.LoggerAdapter,
	router *message.Router,
	appendToTrackerSub message.Subscriber,
	issueReceiptSubscriber message.Subscriber,
	spreadsheetsClient SpreadsheetsService,
	receiptsClient ReceiptsService,
) *EventHandlers {
	eh := &EventHandlers{
		router:                 router,
		appendToTrackerSub:     appendToTrackerSub,
		issueReceiptSubscriber: issueReceiptSubscriber,
		spreadsheetsClient:     spreadsheetsClient,
		receiptsClient:         receiptsClient,
	}

	router.AddMiddleware(middleware.Recoverer)
	router.AddMiddleware(CorrelationIDMiddleware)
	router.AddMiddleware(LoggingMiddleware)
	router.AddMiddleware(MetadataTypeChecker)

	router.AddMiddleware(middleware.Retry{
		MaxRetries:      10,
		InitialInterval: time.Millisecond * 100,
		MaxInterval:     time.Second,
		Multiplier:      2,
		Logger:          wlogger,
	}.Middleware)

	// skip marshalling errors before retrying
	router.AddMiddleware(SkipMarshallingErrorsMiddleware)

	router.AddNoPublisherHandler(
		"ticket_to_print_handler",
		"TicketBookingConfirmed",
		appendToTrackerSub,
		eh.printTicketsHandler,
	)

	router.AddNoPublisherHandler(
		"refund_ticket_handler",
		"TicketBookingCanceled",
		appendToTrackerSub,
		eh.refundTicketsHandler,
	)

	router.AddNoPublisherHandler(
		"issue_receipt_handler",
		"TicketBookingConfirmed",
		issueReceiptSubscriber,
		eh.issueReceiptHandler,
	)

	return eh
}

func (h *EventHandlers) printTicketsHandler(msg *message.Message) error {
	if eventType := msg.Metadata.Get("type"); eventType != "TicketBookingConfirmed" {
		log.FromContext(msg.Context()).
			WithField("type", eventType).
			Warn("Message type not correct")
		return nil
	}

	var payload domain.TicketBookingConfirmedEvent
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		return ErrJsonUnmarshal
	}

	if payload.Price.Currency == "" {
		payload.Price.Currency = "USD"
	}
	return h.spreadsheetsClient.AppendRow(
		msg.Context(), domain.AppendToTrackerRequest{
			SpreadsheetName: "tickets-to-print",
			Rows: []string{
				payload.TicketId,
				payload.CustomerEmail,
				payload.Price.Amount,
				payload.Price.Currency,
			},
		})

}

func (h *EventHandlers) refundTicketsHandler(msg *message.Message) error {
	if eventType := msg.Metadata.Get("type"); eventType != "TicketBookingCanceled" {
		log.FromContext(msg.Context()).
			WithField("type", eventType).
			Warn("Message type not correct")
		return nil
	}

	var payload domain.TicketBookingConfirmedEvent
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		return ErrJsonUnmarshal
	}

	if payload.Price.Currency == "" {
		payload.Price.Currency = "USD"
	}
	return h.spreadsheetsClient.AppendRow(
		msg.Context(),
		domain.AppendToTrackerRequest{
			SpreadsheetName: "tickets-to-refund",
			Rows: []string{
				payload.TicketId,
				payload.CustomerEmail,
				payload.Price.Amount,
				payload.Price.Currency,
			},
		},
	)
}

func (h *EventHandlers) issueReceiptHandler(msg *message.Message) error {
	var payload domain.TicketBookingConfirmedEvent
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		return ErrJsonUnmarshal
	}

	if payload.Price.Currency == "" {
		payload.Price.Currency = "USD"
	}
	_, err = h.receiptsClient.IssueReceipt(
		msg.Context(),
		domain.IssueReceiptRequest{
			TicketID: payload.TicketId,
			Price:    payload.Price,
		})
	return err
}
