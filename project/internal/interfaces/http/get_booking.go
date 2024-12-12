package http

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"net/http"
	"tickets/internal/domain/ops"
	"tickets/internal/repository"
	"time"
)

type OpsBooking struct {
	BookingID uuid.UUID `json:"booking_id"`
	BookedAt  time.Time `json:"booked_at"`

	Tickets map[string]OpsTicket `json:"tickets"`

	LastUpdate time.Time `json:"last_update"`
}

type OpsTicket struct {
	PriceAmount   string `json:"price_amount"`
	PriceCurrency string `json:"price_currency"`
	CustomerEmail string `json:"customer_email"`

	// Status should be set to "confirmed" or "refunded"
	Status string `json:"status"`

	PrintedAt       time.Time `json:"printed_at"`
	PrintedFileName string    `json:"printed_file_name"`

	ReceiptIssuedAt time.Time `json:"receipt_issued_at"`
	ReceiptNumber   string    `json:"receipt_number"`
}

func (s *Server) GetBookingsHandler(c echo.Context) error {
	var (
		bookings []ops.Booking
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

	opsBookings := make([]OpsBooking, 0, len(bookings))
	for _, booking := range bookings {
		opsTickets := make(map[string]OpsTicket, len(booking.Tickets))
		for ticketID, ticket := range booking.Tickets {
			opsTickets[ticketID] = OpsTicket{
				PriceAmount:     ticket.PriceAmount,
				PriceCurrency:   ticket.PriceCurrency,
				CustomerEmail:   ticket.CustomerEmail,
				Status:          ticket.Status,
				PrintedAt:       ticket.PrintedAt,
				PrintedFileName: ticket.PrintedFileName,
				ReceiptIssuedAt: ticket.ReceiptIssuedAt,
				ReceiptNumber:   ticket.ReceiptNumber,
			}
		}

		opsBookings = append(opsBookings, OpsBooking{
			BookingID:  booking.BookingID,
			BookedAt:   booking.BookedAt,
			Tickets:    opsTickets,
			LastUpdate: booking.LastUpdate,
		})
	}

	return c.JSON(http.StatusOK, opsBookings)
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
