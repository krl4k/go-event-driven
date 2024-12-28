package commands

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"tickets/internal/application/usecases/booking"
)

type PaymentsService interface {
	Refund(ctx context.Context, ticketID, idempotencyKey string) error
}

type ReceiptsService interface {
	VoidReceipt(ctx context.Context, ticketID, idempotencyKey string) error
}

type Handler struct {
	eb                 *cqrs.EventBus
	paymentService     PaymentsService
	receiptsService    ReceiptsService
	bookTicketsUsecase *booking.BookTicketsUsecase
}

func NewHandler(
	eb *cqrs.EventBus,
	paymentService PaymentsService,
	receiptsService ReceiptsService,
	bookTicketsUsecase *booking.BookTicketsUsecase,
) *Handler {
	return &Handler{
		eb:                 eb,
		paymentService:     paymentService,
		receiptsService:    receiptsService,
		bookTicketsUsecase: bookTicketsUsecase,
	}
}
