//go:build component

package tests_experiments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"time"
)

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

func (s *ComponentTestSuite) TestTicketStatusConfirmation() {
	ticketID := uuid.NewString()

	fmt.Println("Creating ticket with BookingID: ", ticketID)

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

	requestBody, _ := json.Marshal(testRequest)
	resp, err := s.env.HTTPClient.Post(
		s.env.ServiceURL+"/tickets-status",
		"application/json",
		bytes.NewBuffer(requestBody),
	)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	require.Eventually(s.T(),
		func() bool {
			receipts, err := s.gatewayClients.Receipts.GetReceiptsWithResponse(context.Background())
			require.NoError(s.T(), err)
			require.Equal(s.T(), http.StatusOK, receipts.StatusCode())

			require.NotNil(s.T(), receipts.JSON200, "receipts.JSON200 is nil")

			found := false
			for _, receipt := range *receipts.JSON200 {
				fmt.Printf("receipt.TicketID: %s, %s\n", receipt.TicketId, receipt.Number)

				if receipt.TicketId == ticketID {
					found = true
					break
				}
			}
			return found
		},
		10*time.Second,
		1*time.Second,
	)

	require.Eventually(s.T(),
		func() bool {
			rows, err := s.gatewayClients.Spreadsheets.GetSheetsSheetRowsWithResponse(context.Background(), "tickets-to-print")
			require.NoError(s.T(), err)
			require.Equal(s.T(), http.StatusOK, rows.StatusCode())

			require.NotNil(s.T(), rows.JSON200, "rows.JSON200 is nil")

			found := false
			for _, spreadsheetRows := range rows.JSON200.Rows {
				for _, spreadsheetRow := range spreadsheetRows {
					if spreadsheetRow == ticketID {
						found = true
						break
					}
				}
			}
			return found
		},
		10*time.Second,
		1*time.Second,
	)

}
