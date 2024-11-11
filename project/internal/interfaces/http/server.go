package http

import (
	"context"
	"errors"
	"github.com/labstack/echo/v4"
	"net/http"
	"tickets/internal/application/services"
)

type Server struct {
	e *echo.Echo

	ticketConfirmationService *services.TicketConfirmationService
}

func NewServer(
	e *echo.Echo,
	ticketConfirmationService *services.TicketConfirmationService,
) *Server {
	srv := &Server{
		e:                         e,
		ticketConfirmationService: ticketConfirmationService,
	}
	e.POST("/tickets-confirmation", srv.TicketsConfirmationHandler)

	return srv
}

func (s *Server) Start() error {
	err := s.e.Start(":8080")
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return s.e.Shutdown(ctx)
}
