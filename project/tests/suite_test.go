package tests

import (
	"context"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/golang/mock/gomock"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"net/http"
	"os"
	"testing"
	"tickets/internal/app"
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
