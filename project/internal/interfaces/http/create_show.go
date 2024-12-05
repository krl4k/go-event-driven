package http

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"net/http"
	domain "tickets/internal/domain/shows"
	"time"
)

type CreateShowRequest struct {
	DeadNationId    uuid.UUID `json:"dead_nation_id"`
	NumberOfTickets int       `json:"number_of_tickets"`
	StartTime       time.Time `json:"start_time"`
	Title           string    `json:"title"`
	Venue           string    `json:"venue"`
}

type CreateShowResponse struct {
	ShowID uuid.UUID `json:"show_id"`
}

func (s *Server) CreateShowHandler(c echo.Context) error {
	ctx := c.Request().Context()

	var request CreateShowRequest
	err := c.Bind(&request)
	if err != nil {
		return err
	}

	showID, err := s.showsService.CreateShow(ctx,
		domain.Show{
			DeadNationId:    request.DeadNationId,
			NumberOfTickets: request.NumberOfTickets,
			StartTime:       request.StartTime,
			Title:           request.Title,
			Venue:           request.Venue,
		})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated,
		CreateShowResponse{
			ShowID: showID,
		},
	)
}
