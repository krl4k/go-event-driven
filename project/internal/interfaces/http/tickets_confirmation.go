package http

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

type TicketsConfirmationRequest struct {
	Tickets []string `json:"tickets"`
}

func (s *Server) TicketsConfirmationHandler(ctx echo.Context) error {
	var request TicketsConfirmationRequest
	err := ctx.Bind(&request)
	if err != nil {
		return err
	}

	s.ticketConfirmationService.ConfirmTickets(request.Tickets)

	return ctx.NoContent(http.StatusOK)
}
