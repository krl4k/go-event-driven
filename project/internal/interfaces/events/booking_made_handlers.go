package events

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	domain "tickets/internal/domain/bookings"
	"tickets/internal/infrastructure/clients"
)

func (h *Handler) TicketBookingHandler() cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"ticket_booking_handler",
		func(ctx context.Context, payload *domain.BookingMade) error {
			log.FromContext(ctx).Info("Booking made handler")

			show, err := h.showsRepository.GetShow(ctx, payload.ShowID)
			if err != nil {
				return fmt.Errorf("failed to get show: %w", err)
			}

			err = h.deadNationClient.BookTickets(ctx, clients.TicketBookingRequest{
				BookingId:       payload.BookingID,
				CustomerAddress: payload.CustomerEmail,
				EventId:         show.DeadNationId,
				NumberOfTickets: payload.NumberOfTickets,
			})
			if err != nil {
				return fmt.Errorf("failed to book tickets: %w", err)
			}

			return nil
		})
}
