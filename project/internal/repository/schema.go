package repository

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
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

	_, err = db.ExecContext(context.Background(), `
CREATE TABLE IF NOT EXISTS bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    show_id UUID NOT NULL,
	number_of_tickets INTEGER NOT NULL,
    customer_email VARCHAR(255) NOT NULL,
    CONSTRAINT fk_show 
        FOREIGN KEY (show_id) 
        REFERENCES shows(id)
        ON DELETE RESTRICT
);`)
	if err != nil {
		return fmt.Errorf("failed to create bookings table: %w", err)
	}

	_, err = db.ExecContext(context.Background(), `
CREATE TABLE IF NOT EXISTS read_model_ops_bookings (
	booking_id UUID PRIMARY KEY,
	payload JSONB NOT NULL
);`)
	if err != nil {
		return fmt.Errorf("failed to create read_model_ops_bookings table: %w", err)
	}
	log.FromContext(context.Background()).Info("Database schema initialized")

	return nil
}
