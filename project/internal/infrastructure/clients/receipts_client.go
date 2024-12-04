package clients

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/receipts"
	"net/http"
	domain "tickets/internal/domain/tickets"
)

type ReceiptsClient struct {
	clients *clients.Clients
}

func NewReceiptsClient(clients *clients.Clients) ReceiptsClient {
	return ReceiptsClient{
		clients: clients,
	}
}

func (c ReceiptsClient) IssueReceipt(ctx context.Context, request domain.IssueReceiptRequest) (*domain.IssueReceiptResponse, error) {
	body := receipts.PutReceiptsJSONRequestBody{
		IdempotencyKey: &request.IdempotencyKey,
		TicketId:       request.TicketID,
		Price: receipts.Money{
			MoneyAmount:   request.Price.Amount,
			MoneyCurrency: request.Price.Currency,
		},
	}

	receiptsResp, err := c.clients.Receipts.PutReceiptsWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	if receiptsResp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %v", receiptsResp.StatusCode())
	}

	return &domain.IssueReceiptResponse{
		ReceiptNumber: receiptsResp.JSON200.Number,
		IssuedAt:      receiptsResp.JSON200.IssuedAt,
	}, nil
}
