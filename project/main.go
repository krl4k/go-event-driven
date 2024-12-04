package main

import (
	"context"
	commonClients "github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	nethttp "net/http"
	"os"
	"os/signal"
	"tickets/internal/app"
	"tickets/internal/infrastructure/clients"
)

func main() {
	wlogger := watermill.NewStdLogger(false, false)

	gatewayAddr := os.Getenv("GATEWAY_ADDR")
	redisAddr := os.Getenv("REDIS_ADDR")

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	commonClients, err := commonClients.NewClients(gatewayAddr,
		func(ctx context.Context, req *nethttp.Request) error {
			req.Header.Set("Correlation-ID", log.CorrelationIDFromContext(ctx))
			return nil
		},
	)
	if err != nil {
		panic(err)
	}
	receiptsClient := clients.NewReceiptsClient(commonClients)
	spreadsheetsClient := clients.NewSpreadsheetsClient(commonClients)
	filesClient := clients.NewFilesClient(commonClients)

	db, err := sqlx.Open("postgres", os.Getenv("POSTGRES_URL"))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	a, err := app.NewApp(wlogger, spreadsheetsClient, receiptsClient, filesClient, rdb, db)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	err = a.Run(ctx)
	if err != nil {
		panic(err)
	}
}
