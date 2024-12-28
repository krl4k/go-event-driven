package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"net/http"
	"tickets/internal/application/usecases/booking"
	"tickets/internal/application/usecases/shows"
	"tickets/internal/application/usecases/tickets"
	"tickets/internal/application/usecases/vipbundle"
	"tickets/internal/repository"
)

type Server struct {
	e *echo.Echo

	commandBus              *cqrs.CommandBus
	ticketsService          *tickets.ProcessTicketsUsecase
	showsService            *shows.CreateShowUsecase
	bookingsService         *booking.BookTicketsUsecase
	vipBundleUsecase        *vipbundle.CreateBundleUsecase
	opsBookingReadModelRepo *repository.OpsBookingReadModelRepo
}

func NewServer(
	e *echo.Echo,
	commandBus *cqrs.CommandBus,
	ticketService *tickets.ProcessTicketsUsecase,
	showsService *shows.CreateShowUsecase,
	bookingsService *booking.BookTicketsUsecase,
	opsBookingReadModelRepo *repository.OpsBookingReadModelRepo,
	vipBundleUsecase *vipbundle.CreateBundleUsecase,
) *Server {
	srv := &Server{
		e:                       e,
		commandBus:              commandBus,
		ticketsService:          ticketService,
		showsService:            showsService,
		bookingsService:         bookingsService,
		opsBookingReadModelRepo: opsBookingReadModelRepo,
		vipBundleUsecase:        vipBundleUsecase,
	}
	e.POST("/tickets-status", srv.TicketsStatusHandler)
	e.GET("/tickets", srv.GetTicketsHandler)

	e.PUT("/ticket-refund/:ticket_id", srv.RefundTicketHandler)

	e.POST("/shows", srv.CreateShowHandler)
	e.POST("/book-tickets", srv.BookTicketsHandler)

	e.GET("/ops/bookings", srv.GetBookingsHandler)
	e.GET("/ops/bookings/:booking_id", srv.GetBookingHandler)

	e.POST("/book-vip-bundle", srv.BookVIPBundleHandler)

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

	e.Use(TracingMiddleware())

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

func TracingMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			operationName := fmt.Sprintf("%s %s", req.Method, c.Path())

			// create span
			tracer := otel.Tracer("http")
			ctx, span := tracer.Start(req.Context(), operationName)
			defer span.End()

			// base attributes
			span.SetAttributes(
				attribute.String("http.method", req.Method),
				attribute.String("http.url", req.URL.String()),
				attribute.String("http.path", c.Path()),
				attribute.String("http.host", req.Host),
				attribute.String("http.user_agent", req.UserAgent()),
			)

			// add request id to span
			if requestID := c.Request().Header.Get("X-Request-ID"); requestID != "" {
				span.SetAttributes(attribute.String("http.request_id", requestID))
			}

			if correlationID := c.Request().Header.Get("Correlation-ID"); correlationID != "" {
				span.SetAttributes(attribute.String("http.correlation_id", correlationID))
			}

			// set span to context
			c.SetRequest(req.WithContext(ctx))

			err := next(c)

			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			} else {
				span.SetAttributes(attribute.Int("http.status_code", c.Response().Status))
			}

			return err
		}
	}
}
