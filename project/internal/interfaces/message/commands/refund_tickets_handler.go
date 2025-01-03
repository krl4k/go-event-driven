package commands

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"tickets/internal/entities"
)

func (h *Handler) RefundTicketsHandler() cqrs.CommandHandler {
	return cqrs.NewCommandHandler(
		"refund_tickets",
		func(ctx context.Context, command *entities.RefundTicket) error {
			log.FromContext(ctx).Info("Refunding ticket: ", command.TicketID)

			err := h.paymentService.Refund(ctx, command.TicketID, command.Header.IdempotencyKey)
			if err != nil {
				return fmt.Errorf("error refunding tickets: %w", err)
			}
			log.FromContext(ctx).Info("Payment refunded")

			err = h.receiptsService.VoidReceipt(ctx, command.TicketID, command.Header.IdempotencyKey)
			if err != nil {
				return fmt.Errorf("error voiding receipt: %w", err)
			}
			log.FromContext(ctx).Info("Receipt voided")

			err = h.eb.Publish(ctx, &entities.TicketRefunded_v1{
				Header:   command.Header,
				TicketID: command.TicketID,
			})
			if err != nil {
				return fmt.Errorf("error publishing TicketRefunded_v1 event: %w", err)
			}

			return nil
		},
	)
}
