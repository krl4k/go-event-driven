package events

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"
	"tickets/internal/entities"
	"tickets/internal/infrastructure/clients"
)

//go:generate mockgen -destination=mocks/spreadsheets_service_mock.go -package=mocks . SpreadsheetsService
type SpreadsheetsService interface {
	AppendRow(ctx context.Context, req entities.AppendToTrackerRequest) error
}

//go:generate mockgen -destination=mocks/receipts_service_mock.go -package=mocks . ReceiptsService
type ReceiptsService interface {
	IssueReceipt(ctx context.Context, request entities.IssueReceiptRequest) (*entities.IssueReceiptResponse, error)
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
	Create(ctx context.Context, t *entities.Ticket) error
	Delete(ctx context.Context, ticketID uuid.UUID) error
}

//go:generate mockgen -destination=mocks/shows_repository_mock.go -package=mocks . ShowsRepository
type ShowsRepository interface {
	GetShow(ctx context.Context, id uuid.UUID) (*entities.Show, error)
}

type EventRepository interface {
	SaveEvent(ctx context.Context, event entities.DatalakeEvent) error
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
