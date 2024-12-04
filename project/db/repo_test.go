package repository

import (
	"context"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"sync"
	"testing"
	domain "tickets/internal/domain/tickets"
	"tickets/internal/repository"
)

var db *sqlx.DB
var getDbOnce sync.Once

func getDb() *sqlx.DB {
	getDbOnce.Do(func() {
		var err error
		db, err = sqlx.Open("postgres", os.Getenv("POSTGRES_URL"))
		if err != nil {
			panic(err)
		}
	})
	return db
}

func setupTestDB(t *testing.T) {
	db := getDb()
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS tickets (
            ticket_id UUID PRIMARY KEY,
            price_amount DECIMAL NOT NULL,
            price_currency VARCHAR(3) NOT NULL,
            customer_email VARCHAR(255) NOT NULL
        )
    `)
	require.NoError(t, err)
}

func cleanupTestDB(t *testing.T) {
	db := getDb()
	_, err := db.Exec("TRUNCATE TABLE tickets")
	require.NoError(t, err)
}

func TestTicketsRepo_Create_Integration(t *testing.T) {
	setupTestDB(t)
	t.Cleanup(func() { cleanupTestDB(t) })

	repo := repository.NewTicketsRepo(getDb())
	ctx := context.Background()

	t.Run("successful creation and idempotency", func(t *testing.T) {
		ticketID := uuid.New()
		ticket := &domain.Ticket{
			TicketId:      ticketID.String(),
			CustomerEmail: "test@example.com",
			Price: domain.Money{
				Amount:   "100.00",
				Currency: "USD",
			},
		}

		// First creation
		err := repo.Create(ctx, ticket)
		require.NoError(t, err)

		// Verify the ticket was created
		var count int
		err = getDb().QueryRow("SELECT COUNT(*) FROM tickets WHERE ticket_id = $1", ticketID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Try to create the same ticket again
		err = repo.Create(ctx, ticket)
		require.NoError(t, err)

		// Verify no duplicate was created
		err = getDb().QueryRow("SELECT COUNT(*) FROM tickets WHERE ticket_id = $1", ticketID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Verify all fields were saved correctly
		var savedTicket domain.Ticket
		err = getDb().QueryRow(`
            SELECT ticket_id, price_amount, price_currency, customer_email 
            FROM tickets 
            WHERE ticket_id = $1`, ticketID).
			Scan(
				&savedTicket.TicketId,
				&savedTicket.Price.Amount,
				&savedTicket.Price.Currency,
				&savedTicket.CustomerEmail,
			)

		require.NoError(t, err)
		assert.Equal(t, ticketID.String(), savedTicket.TicketId)
		//assert.Equal(t, "100.00", savedTicket.Price.Amount)
		assert.Equal(t, "USD", savedTicket.Price.Currency)
		assert.Equal(t, "test@example.com", savedTicket.CustomerEmail)
	})

	t.Run("invalid ticket data", func(t *testing.T) {
		invalidTicket := &domain.Ticket{
			TicketId:      "invalid-uuid",
			CustomerEmail: "test@example.com",
			Price: domain.Money{
				Amount:   "100.00",
				Currency: "USD",
			},
		}

		err := repo.Create(ctx, invalidTicket)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to convert domain to model")
	})

	t.Run("concurrent creations", func(t *testing.T) {
		ticketID := uuid.New()
		ticket := &domain.Ticket{
			TicketId:      ticketID.String(),
			CustomerEmail: "concurrent@example.com",
			Price: domain.Money{
				Amount:   "200.00",
				Currency: "EUR",
			},
		}

		// Launch multiple goroutines trying to create the same ticket
		concurrency := 5
		errChan := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				errChan <- repo.Create(ctx, ticket)
			}()
		}

		// Collect all results
		for i := 0; i < concurrency; i++ {
			err := <-errChan
			assert.NoError(t, err)
		}

		// Verify only one ticket was created
		var count int
		err := getDb().QueryRow("SELECT COUNT(*) FROM tickets WHERE ticket_id = $1", ticketID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}
