package commands

import (
	"context"
	"tickets/internal/entities"

	"tickets/internal/infrastructure/clients"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

func (h *Handler) BookFlightHandler() cqrs.CommandHandler {
	return cqrs.NewCommandHandler(
		"book_flight",
		func(ctx context.Context, command *entities.BookFlight) error {
			log.FromContext(ctx).Info("Booking flight")
			resp, err := h.transportationClient.BookFlightTicket(ctx, &clients.BookFlightTicketRequest{
				CustomerEmail:  command.CustomerEmail,
				FlightID:       command.FlightID,
				PassengerNames: command.Passengers,
				ReferenceId:    command.ReferenceID,
				IdempotencyKey: command.IdempotencyKey,
			})
			if err != nil {
				return err
			}
			log.FromContext(ctx).Info("Flight booked", "response", resp)

			err = h.eb.Publish(ctx, &entities.FlightBooked_v1{
				Header:      entities.NewEventHeader(),
				FlightID:    command.FlightID,
				TicketIDs:   resp.TicketsID,
				ReferenceID: command.ReferenceID,
			})
			if err != nil {
				return err
			}

			return nil
		},
	)
}
