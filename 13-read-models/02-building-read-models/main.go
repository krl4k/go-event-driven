package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/shopspring/decimal"
)

type InvoiceIssued struct {
	InvoiceID    string
	CustomerName string
	Amount       decimal.Decimal
	IssuedAt     time.Time
}

type InvoicePaymentReceived struct {
	PaymentID  string
	InvoiceID  string
	PaidAmount decimal.Decimal
	PaidAt     time.Time

	FullyPaid bool
}

type InvoiceVoided struct {
	InvoiceID string
	VoidedAt  time.Time
}

type InvoiceReadModel struct {
	InvoiceID    string
	CustomerName string
	Amount       decimal.Decimal
	IssuedAt     time.Time

	FullyPaid     bool
	PaidAmount    decimal.Decimal
	LastPaymentAt time.Time

	Voided   bool
	VoidedAt time.Time
}

type InvoiceReadModelStorage struct {
	invoices map[string]InvoiceReadModel

	issuedIdempotencyMap  map[string]struct{}
	paymentIdempotencyMap map[string]struct{}
}

func NewInvoiceReadModelStorage() *InvoiceReadModelStorage {
	return &InvoiceReadModelStorage{
		invoices: make(map[string]InvoiceReadModel),

		issuedIdempotencyMap:  make(map[string]struct{}),
		paymentIdempotencyMap: make(map[string]struct{}),
	}
}

func (s *InvoiceReadModelStorage) Invoices() []InvoiceReadModel {
	invoices := make([]InvoiceReadModel, 0, len(s.invoices))
	for _, invoice := range s.invoices {
		invoices = append(invoices, invoice)
	}
	return invoices
}

func (s *InvoiceReadModelStorage) InvoiceByID(id string) (InvoiceReadModel, bool) {
	invoice, ok := s.invoices[id]
	return invoice, ok
}

func (s *InvoiceReadModelStorage) OnInvoiceIssued(ctx context.Context, event *InvoiceIssued) error {
	if _, ok := s.issuedIdempotencyMap[event.InvoiceID]; ok {
		return nil
	}

	s.issuedIdempotencyMap[event.InvoiceID] = struct{}{}

	s.invoices[event.InvoiceID] = InvoiceReadModel{
		InvoiceID:     event.InvoiceID,
		CustomerName:  event.CustomerName,
		Amount:        event.Amount,
		IssuedAt:      event.IssuedAt,
		FullyPaid:     false,
		PaidAmount:    decimal.Decimal{},
		LastPaymentAt: time.Time{},
		Voided:        false,
		VoidedAt:      time.Time{},
	}

	return nil
}

func (s *InvoiceReadModelStorage) OnInvoicePaymentReceived(ctx context.Context, event *InvoicePaymentReceived) error {
	if _, ok := s.paymentIdempotencyMap[event.PaymentID]; ok {
		return nil
	}

	s.paymentIdempotencyMap[event.PaymentID] = struct{}{}

	invoice := s.invoices[event.InvoiceID]

	invoice.PaidAmount = invoice.PaidAmount.Add(event.PaidAmount)
	invoice.LastPaymentAt = event.PaidAt
	invoice.FullyPaid = event.FullyPaid

	s.invoices[event.InvoiceID] = invoice

	return nil
}

func (s *InvoiceReadModelStorage) OnInvoiceVoided(ctx context.Context, event *InvoiceVoided) error {
	invoice := s.invoices[event.InvoiceID]

	invoice.Voided = true
	invoice.VoidedAt = event.VoidedAt

	s.invoices[event.InvoiceID] = invoice

	return nil
}

func NewRouter(storage *InvoiceReadModelStorage, eventProcessorConfig cqrs.EventProcessorConfig, watermillLogger watermill.LoggerAdapter) (*message.Router, error) {
	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return nil, fmt.Errorf("could not create router: %w", err)
	}

	eventProcessor, err := cqrs.NewEventProcessorWithConfig(router, eventProcessorConfig)
	if err != nil {
		return nil, fmt.Errorf("could not create command processor: %w", err)
	}

	err = eventProcessor.AddHandlers(
		cqrs.NewEventHandler(
			"invoice_issued_handler",
			storage.OnInvoiceIssued),

		cqrs.NewEventHandler(
			"invoice_payment_received_handler",
			storage.OnInvoicePaymentReceived),

		cqrs.NewEventHandler(
			"invoice_voided_handler",
			storage.OnInvoiceVoided),
	)
	if err != nil {
		return nil, fmt.Errorf("could not add event handlers: %w", err)
	}

	return router, nil
}
