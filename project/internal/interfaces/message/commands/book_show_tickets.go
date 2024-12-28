package commands

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"tickets/internal/application/usecases/booking"
	"tickets/internal/entities"
)

func (h *Handler) BookShowTicketsHandler() cqrs.CommandHandler {
	return cqrs.NewCommandHandler(
		"book_show_tickets",
		func(ctx context.Context, command *entities.BookShowTickets) error {
			log.FromContext(ctx).Info("Booking tickets for vip bundle")

			_, err := h.bookTicketsUsecase.BookTickets(ctx,
				booking.CreateBookingReq{
					BookingID:       &command.BookingID,
					ShowId:          command.ShowId,
					NumberOfTickets: command.NumberOfTickets,
					CustomerEmail:   command.CustomerEmail,
				})
			if err != nil {
				return fmt.Errorf("book tickets: %w", err)
			}

			return nil
		},
	)
}
