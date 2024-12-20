package entities

import (
	"time"
)

type IssueReceiptRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	TicketID       string `json:"ticket_id"`
	Price          Money  `json:"price"`
}

type IssueReceiptResponse struct {
	ReceiptNumber string    `json:"number"`
	IssuedAt      time.Time `json:"issued_at"`
}
