package events

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"
	"tickets/internal/infrastructure/clients"
)

type AppendToTrackerHandler struct {
	logger   zerolog.Logger
	messages <-chan *message.Message

	spreadSheetsClient clients.SpreadsheetsClient
}

func NewAppendToTrackerHandler(
	logger zerolog.Logger,
	subscriber message.Subscriber,
	spreadsheetsClient clients.SpreadsheetsClient,
) (*AppendToTrackerHandler, error) {
	// todo pass topic name as a param from config
	msg, err := subscriber.Subscribe(context.Background(), "append-to-tracker")
	if err != nil {
		return nil, err
	}

	return &AppendToTrackerHandler{
		logger:             logger,
		messages:           msg,
		spreadSheetsClient: spreadsheetsClient,
	}, nil
}

func (h *AppendToTrackerHandler) Run() {
	for msg := range h.messages {
		ticketID := msg.Payload
		err := h.spreadSheetsClient.AppendRow(context.Background(),
			"tickets-to-print", []string{string(ticketID)})
		if err != nil {
			msg.Nack()
		} else {
			msg.Ack()
		}
	}
}
