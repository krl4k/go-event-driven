package main

import (
	"context"
)

type Task int

const (
	TaskIssueReceipt Task = iota
	TaskAppendToTracker
)

type Message struct {
	Task     Task
	TicketID string
}

type Worker struct {
	queue              chan Message
	receiptsClient     ReceiptsClient
	spreadSheetsClient SpreadsheetsClient
}

func NewWorker(
	receiptsClient ReceiptsClient,
	spreadsheetsClient SpreadsheetsClient,
	n int32,
) *Worker {
	return &Worker{
		receiptsClient:     receiptsClient,
		spreadSheetsClient: spreadsheetsClient,
		queue:              make(chan Message, n),
	}
}

func (w *Worker) Send(msg ...Message) {
	for _, m := range msg {
		w.queue <- m
	}
}

func (w *Worker) Run() {
	for msg := range w.queue {
		var err error
		ctx := context.Background()

		switch msg.Task {
		case TaskIssueReceipt:
			err = w.receiptsClient.IssueReceipt(ctx, msg.TicketID)
		case TaskAppendToTracker:
			err = w.spreadSheetsClient.AppendRow(ctx, "tickets-to-print", []string{msg.TicketID})
		}

		if err != nil {
			w.Send(msg)
		}
	}
}
