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

	_, err = db.ExecContext(context.Background(), `
CREATE TABLE IF NOT EXISTS shows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dead_nation_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    venue VARCHAR(255) NOT NULL,
    number_of_tickets INTEGER NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL
);`)
	if err != nil {
		return fmt.Errorf("failed to create shows table: %w", err)
	}
	return nil
}
