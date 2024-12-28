package http

import (
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/labstack/echo/v4"
	"net/http"
	"tickets/internal/entities"
)

func (s *Server) RefundTicketHandler(ctx echo.Context) error {
	ticketId := ctx.Param("ticket_id")
	if ticketId == "" {
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"reason": "ticket_id is required",
		})
	}

	log.FromContext(ctx.Request().Context()).Info("Refunding ticket: ", ticketId)

	err := s.commandBus.Send(ctx.Request().Context(), &entities.RefundTicket{
		Header:   entities.NewEventHeader(),
		TicketID: ticketId,
	})
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusAccepted)
}
