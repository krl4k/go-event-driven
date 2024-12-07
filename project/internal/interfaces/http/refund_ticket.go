package http

import (
	"github.com/labstack/echo/v4"
	"net/http"
	domain "tickets/internal/domain/tickets"
)

func (s *Server) RefundTicketHandler(ctx echo.Context) error {
	ticketId := ctx.Param("ticket_id")
	if ticketId == "" {
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"reason": "ticket_id is required",
		})
	}

	err := s.commandBus.Send(ctx.Request().Context(), &domain.RefundTicket{
		TicketId: ticketId,
	})
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusAccepted)
}
