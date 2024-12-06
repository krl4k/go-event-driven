package http

import (
	"context"
	"errors"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/labstack/echo/v4"
	"net/http"
	"tickets/internal/application/usecases/booking"
	"tickets/internal/application/usecases/shows"
	"tickets/internal/application/usecases/tickets"
)

type Server struct {
	e *echo.Echo

	ticketsService  *tickets.ProcessTicketsUsecase
	showsService    *shows.CreateShowUsecase
	bookingsService *booking.BookTicketsUsecase
}

func NewServer(
	e *echo.Echo,
	ticketService *tickets.ProcessTicketsUsecase,
	showsService *shows.CreateShowUsecase,
	bookingsService *booking.BookTicketsUsecase,
	routerIsRunning func() bool,
) *Server {
	srv := &Server{
		e:               e,
		ticketsService:  ticketService,
		showsService:    showsService,
		bookingsService: bookingsService,
	}
	e.POST("/tickets-status", srv.TicketsStatusHandler)
	e.GET("/tickets", srv.GetTicketsHandler)

	e.POST("/shows", srv.CreateShowHandler)
	e.POST("/book-tickets", srv.BookTicketsHandler)

	e.GET("/health", func(c echo.Context) error {
		if !routerIsRunning() {
			return c.String(http.StatusServiceUnavailable, "router is not running")
		}
		return c.String(http.StatusOK, "ok")
	})

	// logging middleware
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			log.FromContext(c.Request().Context()).
				WithField("path", c.Request().URL.Path).
				Info("Handling a request")

			err := next(c)

			if err != nil {
				log.FromContext(c.Request().Context()).
					WithField("error", err).
					Error("Request handling error")
			}

			return err
		}
	})
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
