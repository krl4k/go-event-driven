package tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lithammer/shortuuid/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"tickets/internal/app"
	domain "tickets/internal/domain/tickets"
	"tickets/internal/interfaces/events/mocks"
	"time"
)

type ComponentTestSuite struct {
	suite.Suite
	ctrl             *gomock.Controller
	spreadsheetsMock *mocks.MockSpreadsheetsService
	receiptsMock     *mocks.MockReceiptsService
	filesMock        *mocks.MockFileStorageService
	ctx              context.Context
	//redisContainer   testcontainers.Container
	redisClient *redis.Client
	db          *sqlx.DB
	app         *app.App
	httpClient  *http.Client
}

func TestComponentTestSuite(t *testing.T) {
	suite.Run(t, new(ComponentTestSuite))
}

func (suite *ComponentTestSuite) SetupSuite() {
	// Initialize dependencies
	suite.ctrl = gomock.NewController(suite.T())
	suite.spreadsheetsMock = mocks.NewMockSpreadsheetsService(suite.ctrl)
	suite.receiptsMock = mocks.NewMockReceiptsService(suite.ctrl)
	suite.filesMock = mocks.NewMockFileStorageService(suite.ctrl)
	suite.ctx = context.Background()
	suite.httpClient = &http.Client{Timeout: 5 * time.Second}
	var err error

	/*
		// Start Redis container
		req := testcontainers.ContainerRequest{
			Image:        "redis:latest",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		}
		suite.redisContainer, err = testcontainers.GenericContainer(suite.ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		require.NoError(suite.T(), err, "Failed to start Redis container")

		redisAddr, err := suite.redisContainer.MappedPort(suite.ctx, "6379/tcp")
		require.NoError(suite.T(), err, "Failed to map Redis port")
		suite.redisClient = redis.NewClient(&redis.Options{
			Addr: "localhost:" + redisAddr.Port(),
		})
	*/

	suite.redisClient = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})
	// Verify Redis connectivity
	require.NoError(suite.T(), suite.redisClient.Ping(suite.ctx).Err(), "Failed to connect to Redis")

	suite.db = sqlx.MustConnect("postgres", os.Getenv("POSTGRES_URL"))

	// Initialize the app
	suite.app, err = app.NewApp(
		watermill.NopLogger{},
		suite.spreadsheetsMock,
		suite.receiptsMock,
		suite.filesMock,
		suite.redisClient,
		suite.db,
	)
	require.NoError(suite.T(), err, "Failed to initialize the app")

	go func() {
		err := suite.app.Run(suite.ctx)
		if err != nil {
			suite.T().Errorf("App run failed: %v", err)
		}
	}()

	// Wait for the HTTP server to be ready
	waitForHttpServer(suite.T())
}

func waitForHttpServer(t *testing.T) {
	t.Helper()

	require.EventuallyWithT(
		t,
		func(t *assert.CollectT) {
			resp, err := http.Get("http://localhost:8080/health")
			if !assert.NoError(t, err) {
				return
			}
			defer resp.Body.Close()

			if assert.Less(t, resp.StatusCode, 300, "API not ready, http status: %d", resp.StatusCode) {
				return
			}
		},
		time.Second*15,
		time.Millisecond*50,
	)
}

func (suite *ComponentTestSuite) TearDownSuite() {
	// Clean up resources
	//require.NoError(suite.T(), suite.redisContainer.Terminate(suite.ctx), "Failed to terminate Redis container")
	suite.ctrl.Finish()
}

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
