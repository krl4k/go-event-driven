package booking

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	trmsql "github.com/avito-tech/go-transaction-manager/drivers/sql/v2"
	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	trmanager "github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/avito-tech/go-transaction-manager/trm/v2/settings"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"tickets/internal/interfaces/events"
	"tickets/internal/outbox"
	"time"

	bdomain "tickets/internal/domain/bookings"
	sdomain "tickets/internal/domain/shows"
)

//go:generate mockgen -destination=mocks/mock_bookings_repo.go -package=mocks tickets/internal/application/usecases/booking BookingsRepo
type BookingsRepo interface {
	CreateBooking(ctx context.Context, booking bdomain.Booking) (uuid.UUID, error)
	GetBookingsCountByShowID(ctx context.Context, showID uuid.UUID) (int64, error)
}

//go:generate mockgen -destination=mocks/mock_shows_repo.go -package=mocks tickets/internal/application/usecases/booking ShowsRepo
type ShowsRepo interface {
	GetShow(ctx context.Context, id uuid.UUID) (*sdomain.Show, error)
}

type BookTicketsUsecase struct {
	bookingRepo     BookingsRepo
	showsRepo       ShowsRepo
	trManager       *trmanager.Manager
	trGetter        *trmsqlx.CtxGetter
	watermillLogger watermill.LoggerAdapter
}

func NewBookTicketsUsecase(
	bookingRepo BookingsRepo,
	showsRepo ShowsRepo,
	trManager *trmanager.Manager,
	trGetter *trmsqlx.CtxGetter,
	watermillLogger watermill.LoggerAdapter,
) *BookTicketsUsecase {
	return &BookTicketsUsecase{
		bookingRepo:     bookingRepo,
		showsRepo:       showsRepo,
		trManager:       trManager,
		trGetter:        trGetter,
		watermillLogger: watermillLogger,
	}
}
func WithRetry(attempts int, f func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		var lastErr error
		for i := 0; i < attempts; i++ {
			log.FromContext(ctx).Info("retrying, attempt", i+1)

			err := f(ctx)
			if err == nil {
				return nil
			}

			log.FromContext(ctx).Error("failed to execute function", err)
			pgErr := &pq.Error{}
			if errors.As(err, &pgErr); pgErr.Code == "40001" {
				lastErr = err
				continue
			}

			return err
		}
		return lastErr
	}
}

func (s *BookTicketsUsecase) BookTickets(ctx context.Context, booking bdomain.Booking) (uuid.UUID, error) {
	var id uuid.UUID
	var err error
	err = s.trManager.DoWithSettings(
		ctx,
		trmsql.MustSettings(
			settings.Must(settings.WithCancelable(true)),
			trmsql.WithTxOptions(&sql.TxOptions{Isolation: sql.LevelSerializable}),
		),
		func(ctx context.Context) error {
			var show *sdomain.Show
			show, err = s.showsRepo.GetShow(ctx, booking.ShowId)
			if err != nil {
				return fmt.Errorf("failed to get show: %w", err)
			}
			log.FromContext(ctx).Info("show number of tickets: ", show.NumberOfTickets)

			var bookingsCount int64
			bookingsCount, err = s.bookingRepo.GetBookingsCountByShowID(ctx, booking.ShowId)
			if err != nil {
				return fmt.Errorf("failed to get bookings count: %w", err)
			}
			log.FromContext(ctx).Info("bookings count: ", bookingsCount)

			if int(bookingsCount)+booking.NumberOfTickets > show.NumberOfTickets {
				return fmt.Errorf("tickets available: %d, requested: %d, %w", show.NumberOfTickets-int(bookingsCount), booking.NumberOfTickets, bdomain.ErrNotEnoughTickets)
			}

			id, err = s.bookingRepo.CreateBooking(ctx, booking)
			if err != nil {
				return fmt.Errorf("failed to create booking: %w", err)
			}

			tr := s.trGetter.DefaultTrOrDB(ctx, nil)
			if tr == nil {
				return fmt.Errorf("failed to get transaction from context")
			}

			publisher, err := outbox.NewPublisher(tr, s.watermillLogger)
			if err != nil {
				return fmt.Errorf("failed to create event publisher: %w", err)
			}

			eb, err := events.NewEventBus(publisher, s.watermillLogger)
			if err != nil {
				return fmt.Errorf("failed to create event bus: %w", err)
			}

			log.FromContext(ctx).Info("publishing booking made event")
			return eb.Publish(ctx, bdomain.BookingMade{
				BookingID:       id,
				NumberOfTickets: booking.NumberOfTickets,
				CustomerEmail:   booking.CustomerEmail,
				ShowID:          booking.ShowId,
				BookedAt:        time.Now().UTC(),
			})
		})

	return id, err

}
