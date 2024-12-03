package repository

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
)

func InitializeDBSchema(db *sqlx.DB) error {
	_, err := db.ExecContext(context.Background(), `
CREATE TABLE IF NOT EXISTS tickets (
	ticket_id UUID PRIMARY KEY,
	price_amount NUMERIC(10, 2) NOT NULL,
	price_currency CHAR(3) NOT NULL,
	customer_email VARCHAR(255) NOT NULL
);`)
	if err != nil {
		return fmt.Errorf("failed to create tickets table: %w", err)

	}
	return nil
}
