package repository

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"tickets/internal/entities"
)

// temporary datalake using postgres
// should be replaced with a real datalake in prod(google bigquery, aws s3, etc)

type EventsRepository struct {
	db *sqlx.DB
}

func NewEventsRepo(db *sqlx.DB) *EventsRepository {
	return &EventsRepository{db: db}
}

func (r *EventsRepository) SaveEvent(
	ctx context.Context,
	event entities.DatalakeEvent,
) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO events (event_id, published_at, event_name, event_payload)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
	`, event.Id, event.PublishedAt, event.EventName, event.Payload)
	if err != nil {
		return err
	}

	return nil
}

func (r *EventsRepository) GetEvents(ctx context.Context) ([]entities.DatalakeEvent, error) {
	// todo stream events using cursor
	var events []entities.DatalakeEvent
	err := r.db.SelectContext(ctx, &events, `
		SELECT event_id, published_at, event_name, event_payload
		FROM events
	`)
	if err != nil {
		log.Printf("failed to get events: %v\n", err)
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	return events, nil
}

func (r *EventsRepository) IsEmpty(ctx context.Context) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `
		SELECT COUNT(*)
		FROM events
	`)
	if err != nil {
		return false, err
	}

	return count == 0, nil
}
