package tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"net/http"
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
	ctx              context.Context
	redisContainer   testcontainers.Container
	redisClient      *redis.Client
	app              *app.App
	httpClient       *http.Client
}

func TestComponentTestSuite(t *testing.T) {
	suite.Run(t, new(ComponentTestSuite))
}

func (suite *ComponentTestSuite) SetupSuite() {
	// Initialize dependencies
	suite.ctrl = gomock.NewController(suite.T())
	suite.spreadsheetsMock = mocks.NewMockSpreadsheetsService(suite.ctrl)
	suite.receiptsMock = mocks.NewMockReceiptsService(suite.ctrl)
	suite.ctx = context.Background()
	suite.httpClient = &http.Client{Timeout: 5 * time.Second}

	// Start Redis container
	req := testcontainers.ContainerRequest{
		Image:        "redis:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	var err error
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

	// Verify Redis connectivity
	require.NoError(suite.T(), suite.redisClient.Ping(suite.ctx).Err(), "Failed to connect to Redis")

	// Initialize the app
	suite.app, err = app.NewApp(
		watermill.NopLogger{},
		suite.spreadsheetsMock,
		suite.receiptsMock,
		suite.redisClient,
	)
	require.NoError(suite.T(), err, "Failed to initialize the app")

	// Start the app in a separate goroutine
	go func() {
		err := suite.app.Run(suite.ctx)
		if err != nil {
			suite.T().Errorf("App run failed: %v", err)
		}
	}()

	// Wait for the HTTP server to be ready
	waitForHttpServer(suite.T())
}

func (suite *ComponentTestSuite) TearDownSuite() {
	// Clean up resources
	require.NoError(suite.T(), suite.redisContainer.Terminate(suite.ctx), "Failed to terminate Redis container")
	suite.ctrl.Finish()
}

func (suite *ComponentTestSuite) TestCreateTicket() {
	// Test data
	ticketID := uuid.NewString()
	testRequest := TicketsConfirmationRequest{
		Tickets: []Ticket{
			{
				TicketId:      ticketID,
				Status:        "confirmed",
				CustomerEmail: "test1@example.com",
				Price: Price{
					Amount:   "100.00",
					Currency: "USD",
				},
			},
		},
	}
	var receiptsCallCount atomic.Int32
	var spreadsheetsCallCount atomic.Int32

	suite.receiptsMock.EXPECT().
		IssueReceipt(gomock.Any(), gomock.Any()).
		Return(nil, nil).
		Times(1).
		Do(func(context.Context, domain.IssueReceiptRequest) {
			receiptsCallCount.Add(1)
		})

	suite.spreadsheetsMock.EXPECT().
		AppendRow(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1).
		Do(func(context.Context, domain.AppendToTrackerRequest) {
			spreadsheetsCallCount.Add(1)
		})

	// Perform HTTP request
	requestBody, err := json.Marshal(testRequest)
	require.NoError(suite.T(), err, "Failed to marshal test request")

	resp, err := suite.httpClient.Post(
		"http://127.0.0.1:8080/tickets-status",
		"application/json",
		bytes.NewBuffer(requestBody),
	)
	require.NoError(suite.T(), err, "Failed to send HTTP request")
	require.Equal(suite.T(), http.StatusOK, resp.StatusCode, "Unexpected HTTP status code")

	require.Eventually(
		suite.T(),
		func() bool {
			return receiptsCallCount.Load() == 1 && spreadsheetsCallCount.Load() == 1
		},
		5*time.Second,
		100*time.Millisecond,
		"All mocks should have been called",
	)
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

// Supporting Types
type Price struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type Ticket struct {
	TicketId      string `json:"ticket_id"`
	Status        string `json:"status"`
	CustomerEmail string `json:"customer_email"`
	Price         Price  `json:"price"`
}

type TicketsConfirmationRequest struct {
	Tickets []Ticket `json:"tickets"`
}
