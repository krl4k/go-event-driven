package events

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	domain "tickets/internal/domain/tickets"
)

//go:generate mockgen -destination=mocks/spreadsheets_service_mock.go -package=mocks . SpreadsheetsService
type SpreadsheetsService interface {
	AppendRow(ctx context.Context, req domain.AppendToTrackerRequest) error
}

//go:generate mockgen -destination=mocks/receipts_service_mock.go -package=mocks . ReceiptsService
type ReceiptsService interface {
	IssueReceipt(ctx context.Context, request domain.IssueReceiptRequest) (*domain.IssueReceiptResponse, error)
}

func TicketsToPrintHandler(
	spreadsheetsClient SpreadsheetsService,
) cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"ticket_to_print_handler",
		func(ctx context.Context, payload *domain.TicketBookingConfirmed) error {
			if payload.Price.Currency == "" {
				payload.Price.Currency = "USD"
			}
			return spreadsheetsClient.AppendRow(
				ctx, domain.AppendToTrackerRequest{
					SpreadsheetName: "tickets-to-print",
					Rows: []string{
						payload.TicketId,
						payload.CustomerEmail,
						payload.Price.Amount,
						payload.Price.Currency,
					},
				})
		},
	)
}

func RefundTicketHandler(
	spreadsheetsClient SpreadsheetsService,
) cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"refund_ticket_handler",
		func(ctx context.Context, payload *domain.TicketBookingCanceled) error {
			if payload.Price.Currency == "" {
				payload.Price.Currency = "USD"
			}
			return spreadsheetsClient.AppendRow(
				ctx,
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
		},
	)
}

func IssueReceiptHandler(
	receiptsClient ReceiptsService,
) cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"issue_receipt_handler",
		func(ctx context.Context, payload *domain.TicketBookingConfirmed) error {
			if payload.Price.Currency == "" {
				payload.Price.Currency = "USD"
			}
			_, err := receiptsClient.IssueReceipt(
				ctx,
				domain.IssueReceiptRequest{
					TicketID: payload.TicketId,
					Price:    payload.Price,
				})
			return err
		},
	)
}
