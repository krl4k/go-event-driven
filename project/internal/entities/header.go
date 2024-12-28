package entities

import (
	"github.com/google/uuid"
	"time"
)

type EventHeader struct {
	Id             string    `json:"id"`
	PublishedAt    time.Time `json:"published_at"`
	IdempotencyKey string    `json:"idempotency_key"`
}

func NewEventHeader() EventHeader {
	return EventHeader{
		Id:             uuid.NewString(),
		PublishedAt:    time.Now(),
		IdempotencyKey: uuid.NewString(),
	}
}

func NewEventHeaderWithIdempotencyKey(idempotencyKey string) EventHeader {
	return EventHeader{
		Id:             uuid.NewString(),
		PublishedAt:    time.Now(),
		IdempotencyKey: idempotencyKey,
	}
}
