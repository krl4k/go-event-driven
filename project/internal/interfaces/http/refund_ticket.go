package http

import (
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/labstack/echo/v4"
	"net/http"
	domain2 "tickets/internal/domain"
	domain "tickets/internal/domain/tickets"
)

func (s *Server) RefundTicketHandler(ctx echo.Context) error {
	ticketId := ctx.Param("ticket_id")
	if ticketId == "" {
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"reason": "ticket_id is required",
		})
	}

	log.FromContext(ctx.Request().Context()).Info("Refunding ticket: ", ticketId)

	err := s.commandBus.Send(ctx.Request().Context(), &domain.RefundTicket{
		Header:   domain2.NewEventHeader(),
		TicketId: ticketId,
	})
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusAccepted)
}
