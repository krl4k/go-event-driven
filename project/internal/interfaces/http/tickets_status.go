package http

import (
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/labstack/echo/v4"
	"net/http"
	domain "tickets/internal/domain/tickets"
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

func (s *Server) TicketsStatusHandler(ctx echo.Context) error {
	var request TicketsStatusRequest
	err := ctx.Bind(&request)
	if err != nil {
		return err
	}

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

	log.FromContext(ctx.Request().Context()).
		WithField("correlation_id", log.CorrelationIDFromContext(ctx.Request().Context())).
		Info("Confirming tickets http handler")

	s.ticketsService.ProcessTickets(ctx.Request().Context(), tickets)

	return ctx.NoContent(http.StatusOK)
}
