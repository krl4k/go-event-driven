package repository

import (
	"context"
	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"tickets/internal/repository"
	"time"

	bdomain "tickets/internal/domain/bookings"
	tdomain "tickets/internal/domain/tickets"
)

func setupTestReadModelOpsDB(t *testing.T) {
	db := getDb()
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS read_model_ops_bookings (
            booking_id UUID PRIMARY KEY,
            payload JSONB NOT NULL
        )
    `)
	require.NoError(t, err)
}

//func cleanupTestDB(t *testing.T) {
//	db := getDb()
//	_, err := db.Exec("TRUNCATE TABLE read_model_ops_bookings")
//	require.NoError(t, err)
//}

func TestOpsBookingReadModelRepo_Integration(t *testing.T) {
	setupTestReadModelOpsDB(t)
	t.Cleanup(func() { cleanupTestDB(t) })

	trManager := manager.Must(trmsqlx.NewDefaultFactory(db))
	repo := repository.NewOpsBookingReadModelRepo(getDb(), trmsqlx.DefaultCtxGetter, trManager)
	ctx := context.Background()

	t.Run("handle BookingMade event", func(t *testing.T) {
		bookingID := uuid.New()
		bookedAt := time.Now().UTC()

		event := &bdomain.BookingMade{
			BookingID: bookingID,
			BookedAt:  bookedAt,
		}

		err := repo.OnBookingMadeEvent(ctx, event)
		require.NoError(t, err)

		// Verify booking was created
		booking, err := repo.GetByID(ctx, bookingID)
		require.NoError(t, err)
		assert.Equal(t, bookingID, booking.BookingID)
		assert.Equal(t, bookedAt, booking.BookedAt)
		assert.Empty(t, booking.Tickets)

		// Test idempotency
		err = repo.OnBookingMadeEvent(ctx, event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("handle TicketBookingConfirmed_v1 event", func(t *testing.T) {
		bookingID := uuid.New()
		ticketID := uuid.New()
		bookedAt := time.Now().UTC()

		// Create initial booking
		err := repo.OnBookingMadeEvent(ctx, &bdomain.BookingMade{
			BookingID: bookingID,
			BookedAt:  bookedAt,
		})
		require.NoError(t, err)

		// Confirm ticket booking
		event := &tdomain.TicketBookingConfirmed_v1{
			BookingId:     bookingID.String(),
			TicketId:      ticketID.String(),
			CustomerEmail: "test@example.com",
			Price: tdomain.Money{
				Amount:   "100.00",
				Currency: "USD",
			},
		}

		err = repo.OnTicketBookingConfirmedEvent(ctx, event)
		require.NoError(t, err)

		// Verify ticket was added to booking
		booking, err := repo.GetByID(ctx, bookingID)
		require.NoError(t, err)

		ticket, exists := booking.Tickets[ticketID.String()]
		require.True(t, exists)
		assert.Equal(t, "100.00", ticket.PriceAmount)
		assert.Equal(t, "USD", ticket.PriceCurrency)
		assert.Equal(t, "test@example.com", ticket.CustomerEmail)
		assert.Equal(t, "confirmed", ticket.Status)
	})

	t.Run("handle TicketReceiptIssued_v1 event", func(t *testing.T) {
		bookingID := uuid.New()
		ticketID := uuid.New()
		issuedAt := time.Now().UTC()

		// Create initial booking
		err := repo.OnBookingMadeEvent(ctx, &bdomain.BookingMade{
			BookingID: bookingID,
			BookedAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		event := &tdomain.TicketReceiptIssued_v1{
			BookingId:     bookingID.String(),
			TicketId:      ticketID.String(),
			ReceiptNumber: "REC123",
			IssuedAt:      issuedAt,
		}

		err = repo.OnTicketReceiptIssuedEvent(ctx, event)
		require.NoError(t, err)

		booking, err := repo.GetByID(ctx, bookingID)
		require.NoError(t, err)

		ticket, exists := booking.Tickets[ticketID.String()]
		require.True(t, exists)
		assert.Equal(t, "REC123", ticket.ReceiptNumber)
		assert.Equal(t, issuedAt, ticket.ReceiptIssuedAt)
	})

	t.Run("handle TicketRefunded_v1 event", func(t *testing.T) {
		bookingID := uuid.New()
		ticketID := uuid.New()

		// Create initial booking
		err := repo.OnBookingMadeEvent(ctx, &bdomain.BookingMade{
			BookingID: bookingID,
			BookedAt:  time.Now().UTC(),
		})
		require.NoError(t, err)

		// Confirm ticket first
		err = repo.OnTicketBookingConfirmedEvent(ctx, &tdomain.TicketBookingConfirmed_v1{
			BookingId:     bookingID.String(),
			TicketId:      ticketID.String(),
			CustomerEmail: "test@example.com",
			Price:         tdomain.Money{Amount: "100.00", Currency: "USD"},
		})
		require.NoError(t, err)

		// Refund ticket
		event := &tdomain.RefundTicket{
			TicketId: ticketID.String(),
		}

		err = repo.OnTicketRefundedEvent(ctx, event)
		require.NoError(t, err)

		booking, err := repo.GetByTicketID(ctx, ticketID)
		require.NoError(t, err)

		ticket, exists := booking.Tickets[ticketID.String()]
		require.True(t, exists)
		assert.Equal(t, "refunded", ticket.Status)
	})

	//t.Run("GetAll returns all bookings", func(t *testing.T) {
	//	cleanupTestDB(t) // Start fresh
	//
	//	booking1ID := uuid.New()
	//	booking2ID := uuid.New()
	//
	//	// Create two bookings
	//	for _, id := range []uuid.UUID{booking1ID, booking2ID} {
	//		err := repo.OnBookingMadeEvent(ctx, &bdomain.BookingMade{
	//			BookingID: id,
	//			BookedAt:  time.Now().UTC(),
	//		})
	//		require.NoError(t, err)
	//	}
	//
	//	bookings, err := repo.GetAll(ctx)
	//	require.NoError(t, err)
	//	//assert.Len(t, bookings, 2)
	//})
}
