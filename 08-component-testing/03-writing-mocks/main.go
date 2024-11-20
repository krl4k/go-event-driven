package main

import (
	"context"
	"github.com/google/uuid"
	"sync"
	"time"
)

type IssueReceiptRequest struct {
	TicketID string `json:"ticket_id"`
	Price    Money  `json:"price"`
}

type Money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type IssueReceiptResponse struct {
	ReceiptNumber string    `json:"number"`
	IssuedAt      time.Time `json:"issued_at"`
}

type ReceiptsService interface {
	IssueReceipt(ctx context.Context, request IssueReceiptRequest) (IssueReceiptResponse, error)
}

type ReceiptsServiceMock struct {
	lock           sync.Mutex
	IssuedReceipts []IssueReceiptRequest
}

func (m *ReceiptsServiceMock) IssueReceipt(ctx context.Context, request IssueReceiptRequest) (IssueReceiptResponse, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.IssuedReceipts = append(m.IssuedReceipts, request)

	return IssueReceiptResponse{
		ReceiptNumber: uuid.NewString(),
		IssuedAt:      time.Now(),
	}, nil
}
