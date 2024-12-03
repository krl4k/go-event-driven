package events

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"
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

//go:generate mockgen -destination=mocks/tickets_repository_mock.go -package=mocks . TicketsRepository
type TicketsRepository interface {
	Create(ctx context.Context, t *domain.Ticket) error
	Delete(ctx context.Context, ticketID uuid.UUID) error
}

func TicketsToPrintHandler(
	spreadsheetsClient SpreadsheetsService,
) cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"ticket_to_print_handler",
		func(ctx context.Context, payload *domain.TicketBookingConfirmed) error {
			log.FromContext(ctx).Info("Adding ticket to print")

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
			log.FromContext(ctx).Info("Refunding ticket")

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
			log.FromContext(ctx).Info("Issuing receipt")

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

func StoreTicketsHandler(
	ticketsRepository TicketsRepository,
) cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"store_tickets_handler",
		func(ctx context.Context, payload *domain.TicketBookingConfirmed) error {
			log.FromContext(ctx).Info("Storing ticket")

			return ticketsRepository.Create(ctx, &domain.Ticket{
				TicketId:      payload.TicketId,
				Status:        "confirmed",
				CustomerEmail: payload.CustomerEmail,
				Price:         payload.Price,
			})
		},
	)
}

func RemoveTicketsHandler(ticketsRepository TicketsRepository) cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"remove_tickets_handler",
		func(ctx context.Context, payload *domain.TicketBookingCanceled) error {
			log.FromContext(ctx).Info("Removing ticket")

			id, err := uuid.Parse(payload.TicketId)
			if err != nil {
				return fmt.Errorf("failed to parse ticket id: %w", err)
			}

			return ticketsRepository.Delete(ctx, id)
		},
	)
}
