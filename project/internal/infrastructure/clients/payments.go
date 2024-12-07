package clients

import (
	"context"
	"fmt"
	"github.com/AlekSi/pointer"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/payments"
)

type PaymentsClient struct {
	clients *clients.Clients
}

func NewPaymentsClient(clients *clients.Clients) PaymentsClient {
	return PaymentsClient{
		clients: clients,
	}
}

func (c PaymentsClient) Refund(ctx context.Context, ticketID, idempotencyKey string) error {
	resp, err := c.clients.Payments.PutRefundsWithResponse(ctx, payments.PaymentRefundRequest{
		PaymentReference: ticketID,
		Reason:           "customer requested refund",
		DeduplicationId:  pointer.To(idempotencyKey),
	})
	if err != nil {
		return fmt.Errorf("error refunding tickets: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("error refunding tickets: %s", resp.Status())
	}

	return nil
}
