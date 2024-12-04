package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/lithammer/shortuuid/v3"
	"github.com/stretchr/testify/require"
	"net/http"
	"sync/atomic"
	"testing"
	domain "tickets/internal/domain/tickets"
	"time"
)

// Supporting Types
type Price struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type Ticket struct {
	TicketID      string `json:"ticket_id"`
	Status        string `json:"status"`
	CustomerEmail string `json:"customer_email"`
	Price         Price  `json:"price"`
}

type TicketsStatusRequest struct {
	Tickets []Ticket `json:"tickets"`
}

func sendTicketsStatus(t *testing.T,
	idempotencyKey string,
	req TicketsStatusRequest) {
	t.Helper()

	payload, err := json.Marshal(req)
	require.NoError(t, err)

	correlationID := shortuuid.New()

	ticketIDs := make([]string, 0, len(req.Tickets))
	for _, ticket := range req.Tickets {
		ticketIDs = append(ticketIDs, ticket.TicketID)
	}

	httpReq, err := http.NewRequest(
		http.MethodPost,
		"http://localhost:8080/tickets-status",
		bytes.NewBuffer(payload),
	)
	require.NoError(t, err)

	httpReq.Header.Set("Correlation-ID", correlationID)
	httpReq.Header.Set("Idempotency-Key", idempotencyKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func (suite *ComponentTestSuite) TestConfirmedTickets() {
	// Test data
	idempotencyKey := uuid.NewString()
	ticketID := uuid.NewString()
	customerEmail := "test1@example.com"
	status := "confirmed"
	amount := "100.00"
	currency := "USD"
	testRequest := TicketsStatusRequest{
		Tickets: []Ticket{
			{
				TicketID:      ticketID,
				Status:        status,
				CustomerEmail: customerEmail,
				Price: Price{
					Amount:   amount,
					Currency: currency,
				},
			},
		},
	}
	var receiptsCallCount atomic.Int32
	var spreadsheetsCallCount atomic.Int32
	var filesCallCount atomic.Int32

	suite.receiptsMock.EXPECT().
		IssueReceipt(gomock.Any(), domain.IssueReceiptRequest{
			IdempotencyKey: idempotencyKey + ticketID,
			TicketID:       ticketID,
			Price:          domain.Money{Amount: amount, Currency: currency},
		}).
		Return(nil, nil).
		Times(1).
		Do(func(context.Context, domain.IssueReceiptRequest) {
			receiptsCallCount.Add(1)
		})

	suite.spreadsheetsMock.EXPECT().
		AppendRow(gomock.Any(), domain.AppendToTrackerRequest{
			SpreadsheetName: "tickets-to-print",
			Rows: []string{
				ticketID,
				customerEmail,
				amount,
				currency,
			},
		}).
		Return(nil).
		Times(1).
		Do(func(context.Context, domain.AppendToTrackerRequest) {
			spreadsheetsCallCount.Add(1)
		})

	suite.filesMock.EXPECT().Upload(
		gomock.Any(),
		fmt.Sprintf("%s-ticket.html", ticketID),
		gomock.Any(),
	).
		Return(nil).
		Times(1).
		Do(func(_ context.Context, fileID string, _ []byte) {
			filesCallCount.Add(1)
		})

	// Perform request
	sendTicketsStatus(suite.T(), idempotencyKey, testRequest)

	require.Eventually(
		suite.T(),
		func() bool {
			return receiptsCallCount.Load() == 1 &&
				spreadsheetsCallCount.Load() == 1 &&
				filesCallCount.Load() == 1
		},
		5*time.Second,
		100*time.Millisecond,
		"All mocks should have been called",
	)
}

func (suite *ComponentTestSuite) TestIdempotencyConfirmedTickets() {
	// Test data
	idempotencyKey := uuid.NewString()
	ticketID := uuid.NewString()
	customerEmail := "test1@example.com"
	status := "confirmed"
	amount := "100.00"
	currency := "USD"
	testRequest := TicketsStatusRequest{
		Tickets: []Ticket{
			{
				TicketID:      ticketID,
				Status:        status,
				CustomerEmail: customerEmail,
				Price: Price{
					Amount:   amount,
					Currency: currency,
				},
			},
		},
	}
	var receiptsCallCount atomic.Int32
	var spreadsheetsCallCount atomic.Int32
	var filesCallCount atomic.Int32

	suite.receiptsMock.EXPECT().
		IssueReceipt(gomock.Any(), domain.IssueReceiptRequest{
			IdempotencyKey: idempotencyKey + ticketID,
			TicketID:       ticketID,
			Price:          domain.Money{Amount: amount, Currency: currency},
		}).
		Return(nil, nil).
		Times(2).
		Do(func(context.Context, domain.IssueReceiptRequest) {
			receiptsCallCount.Add(1)
		})

	suite.spreadsheetsMock.EXPECT().
		AppendRow(gomock.Any(), domain.AppendToTrackerRequest{
			SpreadsheetName: "tickets-to-print",
			Rows: []string{
				ticketID,
				customerEmail,
				amount,
				currency,
			},
		}).
		Return(nil).
		Times(2).
		Do(func(context.Context, domain.AppendToTrackerRequest) {
			spreadsheetsCallCount.Add(1)
		})

	suite.filesMock.EXPECT().Upload(
		gomock.Any(),
		fmt.Sprintf("%s-ticket.html", ticketID),
		gomock.Any(),
	).
		Return(nil).
		Times(2).
		Do(func(_ context.Context, fileID string, _ []byte) {
			filesCallCount.Add(1)
		})

	// Perform request
	sendTicketsStatus(suite.T(), idempotencyKey, testRequest)
	sendTicketsStatus(suite.T(), idempotencyKey, testRequest)

	require.Eventually(
		suite.T(),
		func() bool {
			return receiptsCallCount.Load() == 2 &&
				spreadsheetsCallCount.Load() == 2 &&
				filesCallCount.Load() == 2
		},
		5*time.Second,
		100*time.Millisecond,
		"All mocks should have been called",
	)
}

func (suite *ComponentTestSuite) TestCancelledTickets() {
	// Test data
	ticketID := uuid.NewString()
	customerEmail := "test1@example.com"
	status := "cancelled"
	amount := "100.00"
	currency := "USD"
	testRequest := TicketsStatusRequest{
		Tickets: []Ticket{
			{
				TicketID:      ticketID,
				Status:        status,
				CustomerEmail: customerEmail,
				Price: Price{
					Amount:   amount,
					Currency: currency,
				},
			},
		},
	}
	var spreadsheetsCallCount atomic.Int32

	suite.spreadsheetsMock.EXPECT().
		AppendRow(gomock.Any(), domain.AppendToTrackerRequest{
			SpreadsheetName: "tickets-to-refund",
			Rows: []string{
				ticketID,
				customerEmail,
				amount,
				currency,
			},
		}).
		Return(nil).
		Times(1).
		Do(func(context.Context, domain.AppendToTrackerRequest) {
			spreadsheetsCallCount.Add(1)
		})

	// Perform request
	sendTicketsStatus(suite.T(), uuid.NewString(), testRequest)

	require.Eventually(
		suite.T(),
		func() bool {
			return spreadsheetsCallCount.Load() == 1
		},
		5*time.Second,
		100*time.Millisecond,
		"All mocks should have been called",
	)
}
