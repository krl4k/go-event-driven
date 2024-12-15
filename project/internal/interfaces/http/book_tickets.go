package http

import (
	"errors"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"net/http"
	"tickets/internal/entities"
)

type BookTicketsRequest struct {
	ShowId          uuid.UUID `json:"show_id"`
	NumberOfTickets int       `json:"number_of_tickets"`
	CustomerEmail   string    `json:"customer_email"`
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

	bookingID := uuid.Nil
	pgErr := &pq.Error{}
	for i := 0; i < 5; i++ {
		bookingID, err = s.bookingsService.BookTickets(ctx,
			entities.Booking{
				ShowId:          request.ShowId,
				NumberOfTickets: request.NumberOfTickets,
				CustomerEmail:   request.CustomerEmail,
			})
		if err != nil {
			if errors.As(err, &pgErr); pgErr.Code == "40001" {
				log.FromContext(ctx).Error("failed to book tickets, retry", err)
				continue
			}
		}
		break
	}
	if err != nil {
		if errors.Is(err, entities.ErrNotEnoughTickets) {
			log.FromContext(ctx).Error("failed to book tickets", err)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"reason": "Not enough tickets available",
			})
		}
		return err
	}

	return c.JSON(http.StatusCreated,
		BookTicketsResponse{
			ID: bookingID,
		},
	)
}
