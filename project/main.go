package main

import (
	"net/http"
	"os"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type TicketsConfirmationRequest struct {
	Tickets []string `json:"tickets"`
}

func main() {
	log.Init(logrus.InfoLevel)

	clients, err := clients.NewClients(os.Getenv("GATEWAY_ADDR"), nil)
	if err != nil {
		panic(err)
	}

	receiptsClient := NewReceiptsClient(clients)
	spreadsheetsClient := NewSpreadsheetsClient(clients)

	worker := NewWorker(receiptsClient, spreadsheetsClient, 10)
	go worker.Run()

	e := commonHTTP.NewEcho()

	e.POST("/tickets-confirmation", func(c echo.Context) error {
		var request TicketsConfirmationRequest
		err := c.Bind(&request)
		if err != nil {
			return err
		}

		for _, ticket := range request.Tickets {
			worker.Send(
				Message{
					Task:     TaskIssueReceipt,
					TicketID: ticket,
				},
				Message{
					Task:     TaskAppendToTracker,
					TicketID: ticket,
				},
			)

			//err = receiptsClient.IssueReceipt(c.Request().Context(), ticket)
			//if err != nil {
			//	return err
			//}

			//err = spreadsheetsClient.AppendRow(c.Request().Context(), "tickets-to-print", []string{ticket})
			//if err != nil {
			//	return err
			//}
		}

		return c.NoContent(http.StatusOK)
	})

	logrus.Info("Server starting...")

	err = e.Start(":8080")
	if err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
