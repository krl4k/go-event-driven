package repository

import (
	"context"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"time"
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
	id uuid.UUID,
	publishedAt time.Time,
	eventName string,
	payload []byte,
) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO events (event_id, published_at, event_name, event_payload)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
	`, id, publishedAt, eventName, payload)
	if err != nil {
		return err
	}

	return nil
}
