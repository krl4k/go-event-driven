package events

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"
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

//go:generate mockgen -destination=mocks/file_storage_service_mock.go -package=mocks . FileStorageService
type FileStorageService interface {
	Upload(ctx context.Context, fileID string, content []byte) error
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

func PrepareTicketsHandler(
	fileStorage FileStorageService,
	eb *cqrs.EventBus,
) cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"prepare_tickets_handler",
		func(ctx context.Context, payload *domain.TicketBookingConfirmed) error {
			log.FromContext(ctx).Info("Preparing ticket")

			if payload.Price.Currency == "" {
				payload.Price.Currency = "USD"
			}

			fileID := fmt.Sprintf("%s-ticket.html", payload.TicketId)
			content := []byte(fmt.Sprintf(`
<!DOCTYPE html>
	<html>
		<head>
			<title>Ticket</title>
		</head>
		<body>
			<h1>Ticket</h1>
			<p>Ticket ID: %s</p>
			<p>Price: %s %s</p>
		</body>	
	</html>
		`, payload.TicketId, payload.Price.Amount, payload.Price.Currency))

			err := fileStorage.Upload(ctx, fileID, content)
			if err != nil {
				return fmt.Errorf("failed to upload ticket: %w", err)
			}

			return eb.Publish(
				ctx,
				domain.TicketPrinted{
					Header: domain.EventHeader{
						Id:          uuid.NewString(),
						PublishedAt: time.Now().Format(time.RFC3339),
					},
					TicketID: payload.TicketId,
					FileName: fileID,
				})
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
					IdempotencyKey: payload.Header.IdempotencyKey,
					TicketID:       payload.TicketId,
					Price:          payload.Price,
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
