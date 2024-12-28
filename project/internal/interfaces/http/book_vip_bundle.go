package http

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"tickets/internal/application/usecases/vipbundle"
)

type vipBundleRequest struct {
	CustomerEmail   string    `json:"customer_email"`
	InboundFlightId uuid.UUID `json:"inbound_flight_id"`
	NumberOfTickets int       `json:"number_of_tickets"`
	Passengers      []string  `json:"passengers"`
	ReturnFlightId  uuid.UUID `json:"return_flight_id"`
	ShowId          uuid.UUID `json:"show_id"`
}

type vipBundleResponse struct {
	BookingId   uuid.UUID `json:"booking_id"`
	VipBundleId uuid.UUID `json:"vip_bundle_id"`
}

func (s *Server) BookVIPBundleHandler(c echo.Context) error {
	var req vipBundleRequest
	if err := c.Bind(&req); err != nil {
		return fmt.Errorf("bind request: %w", err)
	}

	resp, err := s.vipBundleUsecase.CreateBundle(
		c.Request().Context(),
		vipbundle.CreateBundleReq{
			CustomerEmail:   req.CustomerEmail,
			NumberOfTickets: req.NumberOfTickets,
			ShowId:          req.ShowId,
			Passengers:      req.Passengers,
			InboundFlightID: req.InboundFlightId,
			ReturnFlightID:  req.ReturnFlightId,
		},
	)
	if err != nil {
		return fmt.Errorf("book vip bundle: %w", err)
	}

	return c.JSON(201, vipBundleResponse{
		BookingId:   resp.BookingId,
		VipBundleId: resp.VipBundleId,
	})
}
