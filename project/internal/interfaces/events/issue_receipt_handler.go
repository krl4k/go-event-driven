package events

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"
	"tickets/internal/infrastructure/clients"
)

type IssueReceiptHandler struct {
	logger   zerolog.Logger
	messages <-chan *message.Message

	receiptsClient clients.ReceiptsClient
}

func NewIssueReceiptHandler(
	logger zerolog.Logger,
	subscriber message.Subscriber,
	receiptsClient clients.ReceiptsClient,
) (*IssueReceiptHandler, error) {
	// todo pass topic name as a param from config
	msg, err := subscriber.Subscribe(context.Background(), "issue-receipt")
	if err != nil {
		return nil, err
	}

	return &IssueReceiptHandler{
		logger:         logger,
		messages:       msg,
		receiptsClient: receiptsClient,
	}, nil
}

func (h *IssueReceiptHandler) Run() {
	for msg := range h.messages {
		ticketId := msg.Payload
		err := h.receiptsClient.IssueReceipt(context.Background(), string(ticketId))
		if err != nil {
			msg.Nack()
		} else {
			msg.Ack()
		}

	}
}
