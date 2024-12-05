package services

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	trmanager "github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/google/uuid"
	domain "tickets/internal/domain/bookings"
	"tickets/internal/interfaces/events"
	"tickets/internal/outbox"
)

type BookingRepo interface {
	CreateBooking(ctx context.Context, booking domain.Booking) (uuid.UUID, error)
}

type BookingService struct {
	bookingRepo     BookingRepo
	trManager       *trmanager.Manager
	trGetter        *trmsqlx.CtxGetter
	watermillLogger watermill.LoggerAdapter
}

func NewBookingService(
	bookingRepo BookingRepo,
	trManager *trmanager.Manager,
	trGetter *trmsqlx.CtxGetter,
	watermillLogger watermill.LoggerAdapter,
) *BookingService {
	return &BookingService{
		bookingRepo:     bookingRepo,
		trManager:       trManager,
		trGetter:        trGetter,
		watermillLogger: watermillLogger,
	}
}

func (s *BookingService) BookTickets(ctx context.Context, booking domain.Booking) (uuid.UUID, error) {
	var id uuid.UUID

	err := s.trManager.Do(ctx, func(ctx context.Context) error {
		var err error

		id, err = s.bookingRepo.CreateBooking(ctx, booking)
		if err != nil {
			return fmt.Errorf("failed to create booking: %w", err)
		}

		tr := s.trGetter.DefaultTrOrDB(ctx, nil)
		if tr == nil {
			return fmt.Errorf("failed to get transaction from context")
		}

		publisher, err := outbox.NewPublisher(
			tr,
			s.watermillLogger)
		if err != nil {
			return fmt.Errorf("failed to create event publisher: %w", err)
		}

		eb, err := events.NewEventBus(publisher, s.watermillLogger)
		if err != nil {
			return fmt.Errorf("failed to create event bus: %w", err)
		}

		return eb.Publish(ctx, domain.BookingMade{
			BookingID:       id,
			NumberOfTickets: booking.NumberOfTickets,
			CustomerEmail:   booking.CustomerEmail,
			ShowID:          booking.ShowId,
		},
		)
	})

	return id, err
}
