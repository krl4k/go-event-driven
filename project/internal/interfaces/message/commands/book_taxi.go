package commands

import (
	"context"
	"errors"
	"tickets/internal/entities"
	"tickets/internal/infrastructure/clients"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

func (h *Handler) BookTaxiHandler() cqrs.CommandHandler {
	return cqrs.NewCommandHandler(
		"book_taxi",
		func(ctx context.Context, command *entities.BookTaxi) error {
			log.FromContext(ctx).Info("Booking taxi")
			resp, err := h.transportationClient.BookTaxi(ctx, &clients.BookTaxiRequest{
				CustomerEmail:      command.CustomerEmail,
				NumberOfPassengers: command.NumberOfPassengers,
				PassengerName:      command.CustomerName,
				ReferenceId:        command.ReferenceID,
				IdempotencyKey:     command.IdempotencyKey,
			})

			if err != nil {
				log.FromContext(ctx).Info("Error booking taxi", "error", err)

				if errors.Is(err, clients.ErrTaxiAlreadyBooked) {
					// Publish TaxiBookingFailed event
					err := h.eb.Publish(ctx, &entities.TaxiBookingFailed_v1{
						Header:        entities.NewEventHeader(),
						ReferenceID:   command.ReferenceID,
						FailureReason: err.Error(),
					})
					if err != nil {
						return err
					}
					return nil
				}

				return nil // Don't retry on failure
			}

			log.FromContext(ctx).Info("Taxi booked successfully", "response", resp)

			// Publish success event if needed
			err = h.eb.Publish(ctx, &entities.TaxiBooked_v1{
				Header:        entities.NewEventHeader(),
				TaxiBookingID: resp.BookingID,
				ReferenceID:   command.ReferenceID,
			})
			if err != nil {
				return err
			}

			return nil
		},
	)
}
