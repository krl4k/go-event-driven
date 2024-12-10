package http

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

type TicketResponse struct {
	TicketId      string `json:"ticket_id"`
	CustomerEmail string `json:"customer_email"`
	Price         struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	} `json:"price"`
}

func (s *Server) GetTicketsHandler(ctx echo.Context) error {
	tickets, err := s.ticketsService.GetTickets(ctx.Request().Context())
	if err != nil {
		return err
	}

	getTickets := make([]TicketResponse, 0, len(tickets))
	for _, ticket := range tickets {
		getTickets = append(getTickets, TicketResponse{
			TicketId:      ticket.TicketId,
			CustomerEmail: ticket.CustomerEmail,
			Price: struct {
				Amount   string `json:"amount"`
				Currency string `json:"currency"`
			}{
				Amount:   ticket.Price.Amount,
				Currency: ticket.Price.Currency,
			},
		})
	}

	return ctx.JSON(http.StatusOK, getTickets)
}

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

//func (s *Server) GetTicketsHandler(c echo.Context) error {
//	bookings, err := s.opsBookingReadModelRepo.GetAll(c.Request().Context())
//	if err != nil {
//		return c.JSON(http.StatusInternalServerError, err)
//	}
//
//	opsBookings := make([]OpsBooking, 0, len(bookings))
//	for _, booking := range bookings {
//		opsTickets := make(map[string]OpsTicket, len(booking.Tickets))
//		for ticketID, ticket := range booking.Tickets {
//			opsTickets[ticketID] = OpsTicket{
//				PriceAmount:     ticket.PriceAmount,
//				PriceCurrency:   ticket.PriceCurrency,
//				CustomerEmail:   ticket.CustomerEmail,
//				Status:          ticket.Status,
//				PrintedAt:       ticket.PrintedAt,
//				PrintedFileName: ticket.PrintedFileName,
//				ReceiptIssuedAt: ticket.ReceiptIssuedAt,
//				ReceiptNumber:   ticket.ReceiptNumber,
//			}
//		}
//
//		opsBookings = append(opsBookings, OpsBooking{
//			BookingID:  booking.BookingID,
//			BookedAt:   booking.BookedAt,
//			Tickets:    opsTickets,
//			LastUpdate: booking.LastUpdate,
//		})
//	}
//
//	return c.JSON(http.StatusOK, opsBookings)
//}
