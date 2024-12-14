package events

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"
	domain2 "tickets/internal/domain"
	domain "tickets/internal/domain/tickets"
	"time"
)

func (h *Handler) TicketsToPrintHandler() cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"ticket_to_print_handler",
		func(ctx context.Context, payload *domain.TicketBookingConfirmed_v1) error {
			log.FromContext(ctx).Info("Adding ticket to print")

			if payload.Price.Currency == "" {
				payload.Price.Currency = "USD"
			}
			return h.spreadsheetsClient.AppendRow(
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

func (h *Handler) PrepareTicketsHandler() cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"prepare_tickets_handler",
		func(ctx context.Context, payload *domain.TicketBookingConfirmed_v1) error {
			log.FromContext(ctx).Info("Preparing ticket. Generate ticket file HTML")

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

			err := h.fileStorage.Upload(ctx, fileID, content)
			if err != nil {
				return fmt.Errorf("failed to upload ticket: %w", err)
			}

			return h.eb.Publish(
				ctx,
				domain.TicketPrinted_v1{
					Header: domain2.EventHeader{
						Id:          uuid.NewString(),
						PublishedAt: time.Now().Format(time.RFC3339),
					},
					TicketID:  payload.TicketId,
					BookingID: payload.BookingId,
					FileName:  fileID,
					PrintedAt: time.Now().UTC(),
				})
		},
	)
}

func (h *Handler) IssueReceiptHandler() cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"issue_receipt_handler",
		func(ctx context.Context, payload *domain.TicketBookingConfirmed_v1) error {
			log.FromContext(ctx).Info("Issuing receipt")

			if payload.Price.Currency == "" {
				payload.Price.Currency = "USD"
			}
			resp, err := h.receiptsClient.IssueReceipt(
				ctx,
				domain.IssueReceiptRequest{
					IdempotencyKey: payload.Header.IdempotencyKey,
					TicketID:       payload.TicketId,
					Price:          payload.Price,
				})
			if err != nil {
				return fmt.Errorf("failed to issue receipt: %w", err)
			}

			err = h.eb.Publish(
				ctx,
				domain.TicketReceiptIssued_v1{
					Header:        domain2.NewEventHeaderWithIdempotencyKey(payload.Header.IdempotencyKey),
					TicketId:      payload.TicketId,
					ReceiptNumber: resp.ReceiptNumber,
					IssuedAt:      resp.IssuedAt,
					BookingId:     payload.BookingId,
				})
			if err != nil {
				return fmt.Errorf("failed to publish TicketReceiptIssued_v1: %w", err)
			}

			return nil
		},
	)
}

func (h *Handler) StoreTicketsHandler() cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"store_tickets_handler",
		func(ctx context.Context, payload *domain.TicketBookingConfirmed_v1) error {
			log.FromContext(ctx).Info("Storing ticket")

			return h.ticketsRepository.Create(ctx, &domain.Ticket{
				TicketId:      payload.TicketId,
				Status:        "confirmed",
				CustomerEmail: payload.CustomerEmail,
				Price:         payload.Price,
			})
		},
	)
}
