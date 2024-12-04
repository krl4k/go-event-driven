package http

import (
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/labstack/echo/v4"
	"net/http"
	domain "tickets/internal/domain/tickets"
	"tickets/internal/idempotency"
)

type TicketStatusRequest struct {
	TicketId      string `json:"ticket_id"`
	Status        string `json:"status"`
	CustomerEmail string `json:"customer_email"`
	Price         struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	} `json:"price"`
}

type TicketsStatusRequest struct {
	Tickets []TicketStatusRequest `json:"tickets"`
}

func (s *Server) TicketsStatusHandler(c echo.Context) error {
	ctx := c.Request().Context()

	var request TicketsStatusRequest
	err := c.Bind(&request)
	if err != nil {
		return err
	}

	idempotencyKey := c.Request().Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		return c.String(http.StatusBadRequest, "Idenpotency-Key is required")
	}

	ctx = idempotency.WithKey(ctx, idempotencyKey)

	tickets := make([]domain.Ticket, 0, len(request.Tickets))
	for _, ticket := range request.Tickets {
		tickets = append(tickets, domain.Ticket{
			TicketId:      ticket.TicketId,
			Status:        ticket.Status,
			CustomerEmail: ticket.CustomerEmail,
			Price: domain.Money{
				Amount:   ticket.Price.Amount,
				Currency: ticket.Price.Currency,
			},
		})
	}

	log.FromContext(c.Request().Context()).
		WithField("correlation_id", log.CorrelationIDFromContext(c.Request().Context())).
		WithField("idempotency_key", idempotencyKey).
		Info("Confirming tickets http handler")

	s.ticketsService.ProcessTickets(ctx, tickets)

	return c.NoContent(http.StatusOK)
}
