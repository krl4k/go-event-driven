package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"tickets/internal/app"
	"tickets/internal/infrastructure/clients"
	"tickets/internal/observability"

	commonClients "github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file")
	}
	wlogger := watermill.NewStdLogger(false, false)

	gatewayAddr := os.Getenv("GATEWAY_ADDR")
	redisAddr := os.Getenv("REDIS_ADDR")

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	traceHttpClient := &http.Client{Transport: otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("HTTP %s %s %s", r.Method, r.URL.String(), operation)
		}),
	)}

	commonClients, err := commonClients.NewClientsWithHttpClient(gatewayAddr,
		func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Correlation-ID", log.CorrelationIDFromContext(ctx))
			return nil
		},
		traceHttpClient,
	)

	if err != nil {
		panic(err)
	}
	receiptsClient := clients.NewReceiptsClient(commonClients)
	spreadsheetsClient := clients.NewSpreadsheetsClient(commonClients)
	filesClient := clients.NewFilesClient(commonClients)
	deadNationClient := clients.NewDeadNationClient(commonClients)
	paymentsClient := clients.NewPaymentsClient(commonClients)
	transportationClient := clients.NewTransportationClient(commonClients)

	traceDB, err := otelsql.Open("postgres", os.Getenv("POSTGRES_URL"),
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
		otelsql.WithDBName("db"))
	if err != nil {
		panic(err)
	}
	db := sqlx.NewDb(traceDB, "postgres")
	defer db.Close()

	tp := observability.ConfigureTraceProvider()

	a, err := app.NewApp(
		wlogger,
		spreadsheetsClient,
		receiptsClient,
		filesClient,
		deadNationClient,
		paymentsClient,
		transportationClient,
		rdb,
		db,
		tp,
	)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	err = a.Run(ctx)
	if err != nil {
		fmt.Println("Failed to run app: ", err)
	}
}
