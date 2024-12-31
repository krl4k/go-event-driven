package commands

import (
	"context"
	"tickets/internal/entities"

	"tickets/internal/infrastructure/clients"

	"errors"

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
			log.FromContext(ctx).Info("Flight booked: ", "response", resp)
			if err != nil {
				log.FromContext(ctx).Info("Error booking flight", "error", err)

				if errors.Is(err, clients.ErrFlightAlreadyBooked) {
					err := h.eb.Publish(ctx, &entities.FlightBookingFailed_v1{
						Header:        entities.NewEventHeader(),
						FlightID:      command.FlightID,
						ReferenceID:   command.ReferenceID,
						FailureReason: err.Error(),
					})
					if err != nil {
						return err
					}
					// Don't retry if flight is already booked
					return nil
				}
				return err
			}

			//log.FromContext(ctx).Info("Flight booked: ", "response", resp)

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
