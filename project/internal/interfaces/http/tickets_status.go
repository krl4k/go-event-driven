package http

import (
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/labstack/echo/v4"
	"net/http"
	"tickets/internal/entities"
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
	BookingID string `json:"booking_id"`
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
		return c.String(http.StatusBadRequest, "Idempotency-Key is required")
	}

	ctx = idempotency.WithKey(ctx, idempotencyKey)

	tickets := make([]entities.Ticket, 0, len(request.Tickets))
	for _, ticket := range request.Tickets {
		tickets = append(tickets, entities.Ticket{
			TicketId:      ticket.TicketId,
			Status:        ticket.Status,
			CustomerEmail: ticket.CustomerEmail,
			Price: entities.Money{
				Amount:   ticket.Price.Amount,
				Currency: ticket.Price.Currency,
			},
			BookingId: ticket.BookingID,
		})
	}

	log.FromContext(c.Request().Context()).
		WithField("correlation_id", log.CorrelationIDFromContext(c.Request().Context())).
		WithField("idempotency_key", idempotencyKey).
		Info("Confirming tickets http handler")

	err = s.ticketsService.ProcessTickets(ctx, tickets)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"reason": err.Error(),
		})
	}

	return c.NoContent(http.StatusOK)
}
