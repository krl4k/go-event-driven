package entities

import (
	"github.com/google/uuid"
	"time"
)

type DatalakeEvent struct {
	Id          uuid.UUID `db:"event_id"`
	PublishedAt time.Time `db:"published_at"`
	EventName   string    `db:"event_name"`
	Payload     []byte    `db:"event_payload"`
}
