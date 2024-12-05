package events

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"
	sdomain "tickets/internal/domain/shows"
	tdomain "tickets/internal/domain/tickets"
	"tickets/internal/infrastructure/clients"
)

//go:generate mockgen -destination=mocks/spreadsheets_service_mock.go -package=mocks . SpreadsheetsService
type SpreadsheetsService interface {
	AppendRow(ctx context.Context, req tdomain.AppendToTrackerRequest) error
}

//go:generate mockgen -destination=mocks/receipts_service_mock.go -package=mocks . ReceiptsService
type ReceiptsService interface {
	IssueReceipt(ctx context.Context, request tdomain.IssueReceiptRequest) (*tdomain.IssueReceiptResponse, error)
}

//go:generate mockgen -destination=mocks/dead_nation_service_mock.go -package=mocks . DeadNationService
type DeadNationService interface {
	BookTickets(ctx context.Context, request clients.TicketBookingRequest) error
}

//go:generate mockgen -destination=mocks/file_storage_service_mock.go -package=mocks . FileStorageService
type FileStorageService interface {
	Upload(ctx context.Context, fileID string, content []byte) error
}

//go:generate mockgen -destination=mocks/tickets_repository_mock.go -package=mocks . TicketsRepository
type TicketsRepository interface {
	Create(ctx context.Context, t *tdomain.Ticket) error
	Delete(ctx context.Context, ticketID uuid.UUID) error
}

//go:generate mockgen -destination=mocks/shows_repository_mock.go -package=mocks . ShowsRepository
type ShowsRepository interface {
	GetShow(ctx context.Context, id uuid.UUID) (sdomain.Show, error)
}

type Handler struct {
	eb                 *cqrs.EventBus
	spreadsheetsClient SpreadsheetsService
	receiptsClient     ReceiptsService
	fileStorage        FileStorageService
	deadNationClient   DeadNationService
	ticketsRepository  TicketsRepository
	showsRepository    ShowsRepository
}

func NewHandler(
	eb *cqrs.EventBus,
	spreadsheetsClient SpreadsheetsService,
	receiptsClient ReceiptsService,
	fileStorage FileStorageService,
	deadNationClient DeadNationService,
	ticketsRepository TicketsRepository,
	showsRepository ShowsRepository,
) *Handler {
	return &Handler{
		eb:                 eb,
		spreadsheetsClient: spreadsheetsClient,
		receiptsClient:     receiptsClient,
		fileStorage:        fileStorage,
		deadNationClient:   deadNationClient,
		ticketsRepository:  ticketsRepository,
		showsRepository:    showsRepository,
	}
}
