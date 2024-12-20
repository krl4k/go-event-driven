package http

import (
	"context"
	"errors"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"tickets/internal/application/usecases/booking"
	"tickets/internal/application/usecases/shows"
	"tickets/internal/application/usecases/tickets"
	"tickets/internal/repository"
)

type Server struct {
	e *echo.Echo

	commandBus              *cqrs.CommandBus
	ticketsService          *tickets.ProcessTicketsUsecase
	showsService            *shows.CreateShowUsecase
	bookingsService         *booking.BookTicketsUsecase
	opsBookingReadModelRepo *repository.OpsBookingReadModelRepo
}

func NewServer(
	e *echo.Echo,
	commandBus *cqrs.CommandBus,
	ticketService *tickets.ProcessTicketsUsecase,
	showsService *shows.CreateShowUsecase,
	bookingsService *booking.BookTicketsUsecase,
	opsBookingReadModelRepo *repository.OpsBookingReadModelRepo,
) *Server {
	srv := &Server{
		e:                       e,
		commandBus:              commandBus,
		ticketsService:          ticketService,
		showsService:            showsService,
		bookingsService:         bookingsService,
		opsBookingReadModelRepo: opsBookingReadModelRepo,
	}
	e.POST("/tickets-status", srv.TicketsStatusHandler)
	e.GET("/tickets", srv.GetTicketsHandler)

	e.PUT("/ticket-refund/:ticket_id", srv.RefundTicketHandler)

	e.POST("/shows", srv.CreateShowHandler)
	e.POST("/book-tickets", srv.BookTicketsHandler)

	e.GET("/ops/bookings", srv.GetBookingsHandler)
	e.GET("/ops/bookings/:booking_id", srv.GetBookingHandler)

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

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
