package commands

import (
	"context"
	"tickets/internal/entities"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

func (h *Handler) CancelFlightTicketsHandler() cqrs.CommandHandler {
	return cqrs.NewCommandHandler(
		"cancel_flight_tickets",
		func(ctx context.Context, command *entities.CancelFlightTickets) error {

			log.FromContext(ctx).Info("Canceling flight tickets")
			for _, ticketID := range command.FlightTicketIDs { // Assuming FlightTicketIDs is part of the command structure

				err := h.transportationClient.CancelFlightTickets(ctx, ticketID)
				if err != nil {
					log.FromContext(ctx).Info("Error canceling flight tickets", "error", err, "ticketID", ticketID)
				} else {
					log.FromContext(ctx).Info("Successfully canceled flight tickets", "ticketID", ticketID)
				}
			}
			return nil
		},
	)
}
