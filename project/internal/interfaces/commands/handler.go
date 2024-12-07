package commands

import (
	"context"
)

type PaymentsService interface {
	Refund(ctx context.Context, ticketID, idempotencyKey string) error
}

type ReceiptsService interface {
	VoidReceipt(ctx context.Context, ticketID, idempotencyKey string) error
}

type Handler struct {
	paymentService  PaymentsService
	receiptsService ReceiptsService
}

func NewHandler(
	paymentService PaymentsService,
	receiptsService ReceiptsService,
) *Handler {
	return &Handler{
		paymentService:  paymentService,
		receiptsService: receiptsService,
	}
}
