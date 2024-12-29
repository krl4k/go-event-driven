package commands

import (
	"context"
	"tickets/internal/application/usecases/booking"
	"tickets/internal/infrastructure/clients"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

type PaymentsService interface {
	Refund(ctx context.Context, ticketID, idempotencyKey string) error
}

type ReceiptsService interface {
	VoidReceipt(ctx context.Context, ticketID, idempotencyKey string) error
}

type TransportationBooker interface {
	BookTaxi(ctx context.Context, request *clients.BookTaxiRequest) (*clients.BookTaxiResponse, error)
	BookFlightTicket(ctx context.Context, request *clients.BookFlightTicketRequest) (*clients.BookFlightTicketResponse, error)
}

type Handler struct {
	eb                   *cqrs.EventBus
	paymentService       PaymentsService
	receiptsService      ReceiptsService
	bookTicketsUsecase   *booking.BookTicketsUsecase
	transportationClient TransportationBooker
}

func NewHandler(
	eb *cqrs.EventBus,
	paymentService PaymentsService,
	receiptsService ReceiptsService,
	bookTicketsUsecase *booking.BookTicketsUsecase,
	transportationClient TransportationBooker,
) *Handler {
	return &Handler{
		eb:                   eb,
		paymentService:       paymentService,
		receiptsService:      receiptsService,
		bookTicketsUsecase:   bookTicketsUsecase,
		transportationClient: transportationClient,
	}
}
