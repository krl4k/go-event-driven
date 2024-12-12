package http

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

type TicketResponse struct {
	TicketId      string `json:"ticket_id"`
	CustomerEmail string `json:"customer_email"`
	Price         struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	} `json:"price"`
}

func (s *Server) GetTicketsHandler(ctx echo.Context) error {
	tickets, err := s.ticketsService.GetTickets(ctx.Request().Context())
	if err != nil {
		return err
	}

	getTickets := make([]TicketResponse, 0, len(tickets))
	for _, ticket := range tickets {
		getTickets = append(getTickets, TicketResponse{
			TicketId:      ticket.TicketId,
			CustomerEmail: ticket.CustomerEmail,
			Price: struct {
				Amount   string `json:"amount"`
				Currency string `json:"currency"`
			}{
				Amount:   ticket.Price.Amount,
				Currency: ticket.Price.Currency,
			},
		})
	}

	return ctx.JSON(http.StatusOK, getTickets)
}
