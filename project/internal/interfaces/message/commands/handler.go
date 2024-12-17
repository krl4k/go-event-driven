package commands

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

type PaymentsService interface {
	Refund(ctx context.Context, ticketID, idempotencyKey string) error
}

type ReceiptsService interface {
	VoidReceipt(ctx context.Context, ticketID, idempotencyKey string) error
}

type Handler struct {
	eb              *cqrs.EventBus
	paymentService  PaymentsService
	receiptsService ReceiptsService
}

func NewHandler(
	eb *cqrs.EventBus,
	paymentService PaymentsService,
	receiptsService ReceiptsService,
) *Handler {
	return &Handler{
		eb:              eb,
		paymentService:  paymentService,
		receiptsService: receiptsService,
	}
}
