package http

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"net/http"
	domain "tickets/internal/domain/bookings"
)

type BookTicketsRequest struct {
	ShowId          string `json:"show_id"`
	NumberOfTickets int    `json:"number_of_tickets"`
	CustomerEmail   string `json:"customer_email"`
}

type BookTicketsResponse struct {
	ID uuid.UUID `json:"booking_id"`
}

func (s *Server) BookTicketsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	var request BookTicketsRequest
	err := c.Bind(&request)
	if err != nil {
		return err
	}

	bookingID, err := s.bookingsService.BookTickets(ctx,
		domain.Booking{
			ShowId:          request.ShowId,
			NumberOfTickets: request.NumberOfTickets,
			CustomerEmail:   request.CustomerEmail,
		})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated,
		BookTicketsResponse{
			ID: bookingID,
		},
	)
}
