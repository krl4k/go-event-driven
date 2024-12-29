package vipbundle

import (
	"context"
	"database/sql"
	"fmt"
	"tickets/internal/application/usecases/booking"
	"tickets/internal/entities"
	"tickets/internal/interfaces/message/events"
	"tickets/internal/interfaces/message/outbox"

	"github.com/ThreeDotsLabs/watermill"
	trmsql "github.com/avito-tech/go-transaction-manager/drivers/sql/v2"
	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	trmanager "github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/avito-tech/go-transaction-manager/trm/v2/settings"
	"github.com/google/uuid"
)

type Repository interface {
	Add(ctx context.Context, vipBundle entities.VipBundle) error
}

type TicketBooker interface {
	BookTickets(ctx context.Context, req booking.CreateBookingReq) (uuid.UUID, error)
}

type CreateBundleUsecase struct {
	bookTicketsUsecase TicketBooker
	repo               Repository
	trManager          *trmanager.Manager
	trGetter           *trmsqlx.CtxGetter
	watermillLogger    watermill.LoggerAdapter
}

func NewCreateBundleUsecase(
	repo Repository,
	bookTicketsUsecase TicketBooker,
	trManager *trmanager.Manager,
	trGetter *trmsqlx.CtxGetter,
	watermillLogger watermill.LoggerAdapter,
) *CreateBundleUsecase {
	return &CreateBundleUsecase{
		repo:               repo,
		bookTicketsUsecase: bookTicketsUsecase,
		trManager:          trManager,
		trGetter:           trGetter,
		watermillLogger:    watermillLogger,
	}
}

type CreateBundleReq struct {
	CustomerEmail   string
	NumberOfTickets int
	ShowId          uuid.UUID
	Passengers      []string
	InboundFlightID uuid.UUID
	ReturnFlightID  uuid.UUID
}

type CreateBundleRes struct {
	BookingId   uuid.UUID
	VipBundleId uuid.UUID
}

func (u *CreateBundleUsecase) CreateBundle(ctx context.Context, req CreateBundleReq) (*CreateBundleRes, error) {
	vipBundleID := uuid.New()
	bookingID := uuid.New()

	err := u.trManager.DoWithSettings(
		ctx,
		trmsql.MustSettings(
			settings.Must(settings.WithCancelable(true)),
			trmsql.WithTxOptions(&sql.TxOptions{Isolation: sql.LevelSerializable}),
		),
		func(ctx context.Context) error {
			var err error

			bundle := entities.VipBundle{
				VipBundleID:     vipBundleID,
				BookingID:       bookingID,
				CustomerEmail:   req.CustomerEmail,
				NumberOfTickets: req.NumberOfTickets,
				ShowId:          req.ShowId,
				Passengers:      req.Passengers,
				InboundFlightID: req.InboundFlightID,
				ReturnFlightID:  req.ReturnFlightID,
			}

			err = u.repo.Add(ctx, bundle)
			if err != nil {
				return fmt.Errorf("repo vip bundle: %w", err)
			}

			tr := u.trGetter.DefaultTrOrDB(ctx, nil)
			if tr == nil {
				return fmt.Errorf("failed to get transaction from context")
			}

			publisher, err := outbox.NewPublisher(tr, u.watermillLogger)
			if err != nil {
				return fmt.Errorf("failed to create event publisher: %w", err)
			}
			eb, err := events.NewEventBus(publisher, u.watermillLogger)
			if err != nil {
				return fmt.Errorf("failed to create event bus: %w", err)
			}

			err = eb.Publish(ctx, &entities.VipBundleInitialized_v1{
				Header:      entities.NewEventHeader(),
				VipBundleID: bundle.VipBundleID,
			})
			if err != nil {
				return fmt.Errorf("publish vip bundle initialized event: %w", err)
			}

			return nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("create vip bundle: %w", err)
	}

	return &CreateBundleRes{
		BookingId:   bookingID,
		VipBundleId: vipBundleID,
	}, nil
}
