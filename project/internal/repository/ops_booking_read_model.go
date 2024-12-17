package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	trmsql "github.com/avito-tech/go-transaction-manager/drivers/sql/v2"
	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	trmanager "github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/avito-tech/go-transaction-manager/trm/v2/settings"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"tickets/internal/entities"
	"time"
)

type OpsBookingReadModelRepo struct {
	db        *sqlx.DB
	getter    *trmsqlx.CtxGetter
	trManager *trmanager.Manager

	eventBus Publisher
}

//go:generate mockgen -destination=mocks/publisher_mock.go -package=mocks tickets/internal/repository Publisher
type Publisher interface {
	Publish(ctx context.Context, event any) error
}

func NewOpsBookingReadModelRepo(
	db *sqlx.DB,
	getter *trmsqlx.CtxGetter,
	trManager *trmanager.Manager,
	eventBus Publisher,
) *OpsBookingReadModelRepo {
	return &OpsBookingReadModelRepo{
		db:        db,
		getter:    getter,
		trManager: trManager,
		eventBus:  eventBus,
	}
}

func (r *OpsBookingReadModelRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.OpsBooking, error) {
	return r.findReadModelByBookingID(ctx, id.String())
}

func (r *OpsBookingReadModelRepo) GetByTicketID(ctx context.Context, ticketID uuid.UUID) (*entities.OpsBooking, error) {
	return r.findReadModelByTicketID(ctx, ticketID.String())
}

func (r *OpsBookingReadModelRepo) GetAll(ctx context.Context) ([]entities.OpsBooking, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT payload FROM read_model_ops_bookings")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []entities.OpsBooking{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var bookings []entities.OpsBooking
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}

		booking, err := r.unmarshalReadModelFromDB(payload)
		if err != nil {
			return nil, err
		}

		bookings = append(bookings, *booking)
	}

	return bookings, nil
}

type Filters struct {
	ReceiptIssueDate time.Time
}

func (r *OpsBookingReadModelRepo) GetWithFilters(ctx context.Context, filters Filters) ([]entities.OpsBooking, error) {
	query := `
SELECT payload FROM read_model_ops_bookings 
	WHERE booking_id IN (
	    SELECT booking_id FROM (
	        SELECT booking_id, 
	            DATE(jsonb_path_query(payload, '$.tickets.*.receipt_issued_at')::text) as receipt_issued_at 
	        FROM 
	            read_model_ops_bookings
	    ) bookings_within_date 
	    WHERE receipt_issued_at = $1)
`

	rows, err := r.db.QueryContext(ctx, query, filters.ReceiptIssueDate)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var bookings []entities.OpsBooking
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}

		booking, err := r.unmarshalReadModelFromDB(payload)
		if err != nil {
			return nil, err
		}

		bookings = append(bookings, *booking)
	}

	return bookings, nil
}

func (r *OpsBookingReadModelRepo) OnBookingMadeEvent(ctx context.Context, event *entities.BookingMade_v1) error {
	log.FromContext(ctx).Info("OnBookingMadeEvent, event: ", *event)

	return r.trManager.DoWithSettings(
		ctx,
		trmsql.MustSettings(
			settings.Must(settings.WithCancelable(true)),
			trmsql.WithTxOptions(&sql.TxOptions{Isolation: sql.LevelRepeatableRead}),
		),
		func(ctx context.Context) error {
			booking := &entities.OpsBooking{
				BookingID:  event.BookingID,
				BookedAt:   event.BookedAt,
				LastUpdate: time.Now().UTC(),
				Tickets:    nil,
			}

			payload, err := json.Marshal(booking)
			if err != nil {
				return err
			}
			res, err := r.db.ExecContext(ctx, `
		INSERT INTO read_model_ops_bookings (booking_id, payload)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, booking.BookingID, payload)
			if err != nil {
				return err
			}
			if rowsAffected, _ := res.RowsAffected(); rowsAffected == 0 {
				return fmt.Errorf("booking with id %s already exists", booking.BookingID)
			}

			return r.eventBus.Publish(ctx, &entities.InternalOpsReadModelUpdated{
				Header:    entities.NewEventHeader(),
				BookingID: booking.BookingID,
			})
		},
	)

}

func (r *OpsBookingReadModelRepo) OnBookingMadeV0Event(ctx context.Context, event *entities.BookingMade_v0) error {
	log.FromContext(ctx).Info("OnBookingMadeEvent, event: ", *event)

	return r.trManager.DoWithSettings(
		ctx,
		trmsql.MustSettings(
			settings.Must(settings.WithCancelable(true)),
			trmsql.WithTxOptions(&sql.TxOptions{Isolation: sql.LevelRepeatableRead}),
		),
		func(ctx context.Context) error {
			bookedAt, err := time.Parse(time.RFC3339, event.Header.PublishedAt)
			if err != nil {
				return fmt.Errorf("failed to parse booked at time: %w", err)
			}

			booking := &entities.OpsBooking{
				BookingID:  event.BookingID,
				BookedAt:   bookedAt,
				LastUpdate: time.Now().UTC(),
				Tickets:    nil,
			}

			payload, err := json.Marshal(booking)
			if err != nil {
				return err
			}
			res, err := r.db.ExecContext(ctx, `
		INSERT INTO read_model_ops_bookings (booking_id, payload)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, booking.BookingID, payload)
			if err != nil {
				return err
			}
			if rowsAffected, _ := res.RowsAffected(); rowsAffected == 0 {
				return fmt.Errorf("booking with id %s already exists", booking.BookingID)
			}

			return r.eventBus.Publish(ctx, &entities.InternalOpsReadModelUpdated{
				Header:    entities.NewEventHeader(),
				BookingID: booking.BookingID,
			})
		},
	)

}

func (r *OpsBookingReadModelRepo) OnTicketBookingConfirmedEvent(ctx context.Context, event *entities.TicketBookingConfirmed_v1) error {
	return r.trManager.DoWithSettings(
		ctx,
		trmsql.MustSettings(
			settings.Must(settings.WithCancelable(true)),
			trmsql.WithTxOptions(&sql.TxOptions{Isolation: sql.LevelRepeatableRead}),
		),
		func(ctx context.Context) error {
			findReadModelByBookingID, err := r.findReadModelByBookingID(ctx, event.BookingId)
			if err != nil {
				return fmt.Errorf("OnTicketBookingConfirmedEvent. failed to find read model by booking ID: %w", err)
			}

			if findReadModelByBookingID == nil {
				return nil
			}

			ticket := findReadModelByBookingID.Tickets[event.TicketId]

			confirmedAt, err := time.Parse(time.RFC3339, event.Header.PublishedAt)
			if err != nil {
				return fmt.Errorf("failed to parse confirmed at time: %w", err)
			}

			ticket.ConfirmedAt = confirmedAt
			ticket.PriceAmount = event.Price.Amount
			ticket.PriceCurrency = event.Price.Currency
			ticket.CustomerEmail = event.CustomerEmail

			findReadModelByBookingID.Tickets[event.TicketId] = ticket

			err = r.updateReadModel(ctx, findReadModelByBookingID)
			if err != nil {
				return fmt.Errorf("failed to update read model: %w", err)
			}

			return nil
		},
	)
}

func (r *OpsBookingReadModelRepo) OnTicketReceiptIssuedEvent(ctx context.Context, event *entities.TicketReceiptIssued_v1) error {
	log.FromContext(ctx).Info("OnTicketReceiptIssuedEvent", "event:", *event)

	return r.trManager.DoWithSettings(
		ctx,
		trmsql.MustSettings(
			settings.Must(settings.WithCancelable(true)),
			trmsql.WithTxOptions(&sql.TxOptions{Isolation: sql.LevelRepeatableRead}),
		),
		func(ctx context.Context) error {
			findReadModelByBookingID, err := r.findReadModelByBookingID(ctx, event.BookingId)
			if err != nil {
				return fmt.Errorf("OnTicketReceiptIssuedEvent. failed to find read model by booking ID: %w", err)
			}

			if findReadModelByBookingID == nil {
				return nil
			}

			ticket, ok := findReadModelByBookingID.Tickets[event.TicketId]
			if !ok {
				return fmt.Errorf("ticket with id %s not found in booking with id %s", event.TicketId, event.BookingId)
			}

			ticket.ReceiptNumber = event.ReceiptNumber
			ticket.ReceiptIssuedAt = event.IssuedAt

			findReadModelByBookingID.Tickets[event.TicketId] = ticket

			err = r.updateReadModel(ctx, findReadModelByBookingID)
			if err != nil {
				return fmt.Errorf("failed to update read model: %w", err)
			}

			return nil
		},
	)
}

func (r *OpsBookingReadModelRepo) OnTicketReceiptIssuedV0Event(ctx context.Context, event *entities.TicketReceiptIssued_v0) error {
	log.FromContext(ctx).Info("OnTicketReceiptIssuedEvent", "event:", *event)

	return r.trManager.DoWithSettings(
		ctx,
		trmsql.MustSettings(
			settings.Must(settings.WithCancelable(true)),
			trmsql.WithTxOptions(&sql.TxOptions{Isolation: sql.LevelRepeatableRead}),
		),
		func(ctx context.Context) error {
			findReadModelByTicketID, err := r.findReadModelByTicketID(ctx, event.TicketId)
			if err != nil {
				return fmt.Errorf("OnTicketReceiptIssuedEvent. failed to find read model by booking ID: %w", err)
			}

			ticket, ok := findReadModelByTicketID.Tickets[event.TicketId]
			if !ok {
				return fmt.Errorf("ticket with id %s not found", event.TicketId)
			}

			ticket.ReceiptNumber = event.ReceiptNumber
			ticket.ReceiptIssuedAt = event.IssuedAt

			findReadModelByTicketID.Tickets[event.TicketId] = ticket

			err = r.updateReadModel(ctx, findReadModelByTicketID)
			if err != nil {
				return fmt.Errorf("failed to update read model: %w", err)
			}

			return nil
		},
	)
}

func (r *OpsBookingReadModelRepo) OnTicketPrintedEvent(ctx context.Context, event *entities.TicketPrinted_v1) error {
	log.FromContext(ctx).Info("OnTicketPrintedEvent, ticketID:", event.TicketID, ", bookingID:", event.BookingID)

	return r.trManager.DoWithSettings(
		ctx,
		trmsql.MustSettings(
			settings.Must(settings.WithCancelable(true)),
			trmsql.WithTxOptions(&sql.TxOptions{Isolation: sql.LevelRepeatableRead}),
		),
		func(ctx context.Context) error {
			findReadModelByTicketID, err := r.findReadModelByBookingID(ctx, event.BookingID)
			if err != nil {
				log.FromContext(ctx).Error("OnTicketPrintedEvent. failed to find read model by booking ID: ", event.BookingID, " error:", err)
				return fmt.Errorf("OnTicketPrintedEvent. failed to find read model by booking ID: %w", err)
			}

			if findReadModelByTicketID == nil {
				return nil
			}

			ticket, ok := findReadModelByTicketID.Tickets[event.TicketID]
			if !ok {
				log.FromContext(ctx).Error("ticket with id not found in booking", "ticketID:", event.TicketID, "bookingID:", event.BookingID)
				return fmt.Errorf("ticket with id %s not found in booking with id %s", event.TicketID, event.BookingID)
			}

			ticket.PrintedAt = event.PrintedAt
			ticket.PrintedFileName = event.FileName

			findReadModelByTicketID.Tickets[event.TicketID] = ticket

			err = r.updateReadModel(ctx, findReadModelByTicketID)
			if err != nil {
				log.FromContext(ctx).Error("failed to update read model", "error:", err)
				return fmt.Errorf("failed to update read model: %w", err)
			}

			return nil
		},
	)
}

func (r *OpsBookingReadModelRepo) OnTicketPrintedV0Event(ctx context.Context, event *entities.TicketPrinted_v0) error {
	log.FromContext(ctx).Info("OnTicketPrintedV0Event, ticketID:", event.TicketID)

	return r.trManager.DoWithSettings(
		ctx,
		trmsql.MustSettings(
			settings.Must(settings.WithCancelable(true)),
			trmsql.WithTxOptions(&sql.TxOptions{Isolation: sql.LevelRepeatableRead}),
		),
		func(ctx context.Context) error {
			findReadModelByTicketID, err := r.findReadModelByTicketID(ctx, event.TicketID)
			if err != nil {
				log.FromContext(ctx).Error("OnTicketPrintedEvent. failed to find read model by ticket ID: ", event.TicketID, ", ", " error:", err)
				return fmt.Errorf("OnTicketPrintedEvent. failed to find read model by booking ID: %w", err)
			}

			ticket, ok := findReadModelByTicketID.Tickets[event.TicketID]
			if !ok {
				log.FromContext(ctx).Error("ticket with id not found in booking", "ticketID:", event.TicketID)
				return fmt.Errorf("ticket with id %s not found", event.TicketID)
			}

			printedAt, err := time.Parse(time.RFC3339, event.Header.PublishedAt)
			if err != nil {
				return fmt.Errorf("failed to parse printed at time: %w", err)
			}

			ticket.PrintedAt = printedAt
			ticket.PrintedFileName = event.FileName

			findReadModelByTicketID.Tickets[event.TicketID] = ticket

			err = r.updateReadModel(ctx, findReadModelByTicketID)
			if err != nil {
				log.FromContext(ctx).Error("failed to update read model", "error:", err)
				return fmt.Errorf("failed to update read model: %w", err)
			}

			return nil
		},
	)
}

func (r *OpsBookingReadModelRepo) OnTicketRefundedEvent(ctx context.Context, event *entities.TicketRefunded_v1) error {
	log.FromContext(ctx).Info("OnTicketRefundedEvent", "event:", *event)

	return r.trManager.DoWithSettings(
		ctx,
		trmsql.MustSettings(
			settings.Must(settings.WithCancelable(true)),
			trmsql.WithTxOptions(&sql.TxOptions{Isolation: sql.LevelRepeatableRead}),
		),
		func(ctx context.Context) error {
			findReadModelByTicketID, err := r.findReadModelByTicketID(ctx, event.TicketID)
			if err != nil {
				return fmt.Errorf("failed to find read model by ticket ID: %w", err)
			}

			ticket, ok := findReadModelByTicketID.Tickets[event.TicketID]
			if !ok {
				return fmt.Errorf("ticket with id %s not found in booking with id %s", event.TicketID, findReadModelByTicketID.BookingID)
			}

			refundedAt, err := time.Parse(time.RFC3339, event.Header.PublishedAt)
			if err != nil {
				return fmt.Errorf("failed to parse confirmed at time: %w", err)
			}

			ticket.ConfirmedAt = refundedAt
			findReadModelByTicketID.Tickets[event.TicketID] = ticket

			err = r.updateReadModel(ctx, findReadModelByTicketID)
			if err != nil {
				return fmt.Errorf("failed to update read model: %w", err)
			}

			return nil
		},
	)
}

func (r *OpsBookingReadModelRepo) findReadModelByBookingID(
	ctx context.Context,
	bookingID string,
) (*entities.OpsBooking, error) {
	id, err := uuid.Parse(bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ID: %w", err)
	}

	var payload []byte

	err = r.getter.DefaultTrOrDB(ctx, r.db).QueryRowContext(
		ctx,
		"SELECT payload FROM read_model_ops_bookings WHERE booking_id = $1",
		id,
	).Scan(&payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if payload == nil {
		return nil, fmt.Errorf("booking with id %s not found", bookingID)
	}

	return r.unmarshalReadModelFromDB(payload)
}

func (r *OpsBookingReadModelRepo) findReadModelByTicketID(
	ctx context.Context,
	ticketID string,
) (*entities.OpsBooking, error) {
	id, err := uuid.Parse(ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ID: %w", err)
	}

	var payload []byte

	err = r.getter.DefaultTrOrDB(ctx, r.db).QueryRowContext(
		ctx,
		"SELECT payload FROM read_model_ops_bookings WHERE payload::jsonb -> 'tickets' ? $1",
		id,
	).Scan(&payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return r.unmarshalReadModelFromDB(payload)
}

func (r *OpsBookingReadModelRepo) updateReadModel(ctx context.Context, readModel *entities.OpsBooking) error {
	payload, err := r.marshalReadModelToDB(readModel)
	if err != nil {
		return err
	}

	_, err = r.getter.DefaultTrOrDB(ctx, r.db).ExecContext(
		ctx,
		"UPDATE read_model_ops_bookings SET payload = $1 WHERE booking_id = $2",
		payload,
		readModel.BookingID,
	)
	if err != nil {
		return err
	}

	return r.eventBus.Publish(ctx, &entities.InternalOpsReadModelUpdated{
		Header:    entities.NewEventHeader(),
		BookingID: readModel.BookingID,
	})
}

func (r *OpsBookingReadModelRepo) marshalReadModelToDB(readModel *entities.OpsBooking) ([]byte, error) {
	return json.Marshal(readModel)
}

func (r *OpsBookingReadModelRepo) unmarshalReadModelFromDB(payload []byte) (*entities.OpsBooking, error) {
	var dbReadModel entities.OpsBooking
	if err := json.Unmarshal(payload, &dbReadModel); err != nil {
		return nil, err
	}

	if dbReadModel.Tickets == nil {
		dbReadModel.Tickets = map[string]entities.OpsTicket{}
	}

	return &dbReadModel, nil
}

func (r *OpsBookingReadModelRepo) OnTicketBookingConfirmedV0Event(ctx context.Context, event *entities.TicketBookingConfirmed_v0) error {
	return r.trManager.DoWithSettings(
		ctx,
		trmsql.MustSettings(
			settings.Must(settings.WithCancelable(true)),
			trmsql.WithTxOptions(&sql.TxOptions{Isolation: sql.LevelRepeatableRead}),
		),
		func(ctx context.Context) error {
			findReadModelByBookingID, err := r.findReadModelByBookingID(ctx, event.BookingId)
			if err != nil {
				return fmt.Errorf("OnTicketBookingConfirmedV0Event. failed to find read model by booking ID: %w", err)
			}

			ticket := findReadModelByBookingID.Tickets[event.TicketId]

			confirmedAt, err := time.Parse(time.RFC3339, event.Header.PublishedAt)
			if err != nil {
				return fmt.Errorf("failed to parse confirmed at time: %w", err)
			}

			ticket.ConfirmedAt = confirmedAt
			ticket.PriceAmount = event.Price.Amount
			ticket.PriceCurrency = event.Price.Currency
			ticket.CustomerEmail = event.CustomerEmail

			findReadModelByBookingID.Tickets[event.TicketId] = ticket

			err = r.updateReadModel(ctx, findReadModelByBookingID)
			if err != nil {
				return fmt.Errorf("failed to update read model: %w", err)
			}

			return nil
		},
	)
}

func (r *OpsBookingReadModelRepo) OnTicketBookingCanceledV0Event(ctx context.Context, event *entities.TicketBookingCanceled_v0) error {
	return r.trManager.DoWithSettings(
		ctx,
		trmsql.MustSettings(
			settings.Must(settings.WithCancelable(true)),
			trmsql.WithTxOptions(&sql.TxOptions{Isolation: sql.LevelRepeatableRead}),
		),
		func(ctx context.Context) error {
			findReadModelByBookingID, err := r.findReadModelByBookingID(ctx, event.BookingId)
			if err != nil {
				return fmt.Errorf("OnTicketBookingCanceledV0Event. failed to find read model by booking ID: %w", err)
			}

			ticket := findReadModelByBookingID.Tickets[event.TicketId]

			findReadModelByBookingID.Tickets[event.TicketId] = ticket

			err = r.updateReadModel(ctx, findReadModelByBookingID)
			if err != nil {
				return fmt.Errorf("failed to update read model: %w", err)
			}

			return nil
		},
	)
}

func (r *OpsBookingReadModelRepo) OnTicketRefundedV0Event(ctx context.Context, event *entities.TicketRefunded_v0) error {
	log.FromContext(ctx).Info("OnTicketRefundedV0Event", "event:", *event)

	return r.trManager.DoWithSettings(
		ctx,
		trmsql.MustSettings(
			settings.Must(settings.WithCancelable(true)),
			trmsql.WithTxOptions(&sql.TxOptions{Isolation: sql.LevelRepeatableRead}),
		),
		func(ctx context.Context) error {
			findReadModelByTicketID, err := r.findReadModelByTicketID(ctx, event.TicketID)
			if err != nil {
				return fmt.Errorf("failed to find read model by ticket ID: %w", err)
			}

			ticket, ok := findReadModelByTicketID.Tickets[event.TicketID]
			if !ok {
				return fmt.Errorf("ticket with id %s not found", event.TicketID)
			}

			refundedAt, err := time.Parse(time.RFC3339, event.Header.PublishedAt)
			if err != nil {
				return fmt.Errorf("failed to parse refunded at time: %w", err)
			}

			ticket.RefundedAt = refundedAt
			findReadModelByTicketID.Tickets[event.TicketID] = ticket

			err = r.updateReadModel(ctx, findReadModelByTicketID)
			if err != nil {
				return fmt.Errorf("failed to update read model: %w", err)
			}

			return nil
		},
	)
}
