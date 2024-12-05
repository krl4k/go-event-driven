package shows

import (
	"github.com/google/uuid"
	"time"
)

type Show struct {
	Id              uuid.UUID `json:"show_id"`
	DeadNationId    uuid.UUID `json:"dead_nation_id"`
	NumberOfTickets int       `json:"number_of_tickets"`
	StartTime       time.Time `json:"start_time"`
	Title           string    `json:"title"`
	Venue           string    `json:"venue"`
}
