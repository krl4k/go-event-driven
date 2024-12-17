package events

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"
	"tickets/internal/entities"
)

func (h *Handler) RefundTicketHandler() cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"refund_ticket_handler",
		func(ctx context.Context, payload *entities.TicketBookingCanceled_v1) error {
			log.FromContext(ctx).Info("Refunding ticket")

			if payload.Price.Currency == "" {
				payload.Price.Currency = "USD"
			}
			return h.spreadsheetsClient.AppendRow(
				ctx,
				entities.AppendToTrackerRequest{
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

func (h *Handler) RemoveTicketsHandler() cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"remove_tickets_handler",
		func(ctx context.Context, payload *entities.TicketBookingCanceled_v1) error {
			log.FromContext(ctx).Info("Removing ticket")

			id, err := uuid.Parse(payload.TicketId)
			if err != nil {
				return fmt.Errorf("failed to parse ticket id: %w", err)
			}

			return h.ticketsRepository.Delete(ctx, id)
		},
	)
}
