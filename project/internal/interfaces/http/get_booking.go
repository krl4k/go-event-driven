package http

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"net/http"
	"tickets/internal/entities"
	"tickets/internal/repository"
	"time"
)

func (s *Server) GetBookingsHandler(c echo.Context) error {
	var (
		bookings []entities.OpsBooking
	)
	receiptIssueDate := c.QueryParam("receipt_issue_date")
	if receiptIssueDate != "" {
		var err error
		issueDate, err := time.Parse("2006-01-02", receiptIssueDate)
		if err != nil {
			return c.JSON(http.StatusBadRequest, "receipt_issue_date is not a valid date")
		}

		bookings, err = s.opsBookingReadModelRepo.GetWithFilters(c.Request().Context(),
			repository.Filters{
				ReceiptIssueDate: issueDate,
			})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}
	} else {
		var err error
		bookings, err = s.opsBookingReadModelRepo.GetAll(c.Request().Context())
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}
	}

	return c.JSON(http.StatusOK, bookings)
}

func (s *Server) GetBookingHandler(c echo.Context) error {
	bookingID := c.Param("booking_id")
	if bookingID == "" {
		return c.JSON(http.StatusBadRequest, "booking_id is required")
	}

	id, err := uuid.Parse(bookingID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "booking_id is not a valid UUID")
	}

	booking, err := s.opsBookingReadModelRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	if booking == nil {
		return c.JSON(http.StatusNotFound, "booking not found")
	}

	return c.JSON(http.StatusOK, booking)
}
