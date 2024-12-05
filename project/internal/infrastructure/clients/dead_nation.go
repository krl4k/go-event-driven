package clients

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/dead_nation"
	"github.com/google/uuid"
)

type DeadNationClient struct {
	clients *clients.Clients
}

func NewDeadNationClient(clients *clients.Clients) DeadNationClient {
	return DeadNationClient{
		clients: clients,
	}
}

type TicketBookingRequest struct {
	BookingId       uuid.UUID
	CustomerAddress string
	EventId         uuid.UUID
	NumberOfTickets int
}

func (c DeadNationClient) BookTickets(ctx context.Context, request TicketBookingRequest) error {
	resp, err := c.clients.DeadNation.PostTicketBookingWithResponse(ctx, dead_nation.PostTicketBookingRequest{
		BookingId:       request.BookingId,
		CustomerAddress: request.CustomerAddress,
		EventId:         request.EventId,
		NumberOfTickets: request.NumberOfTickets,
	})
	if err != nil {
		return fmt.Errorf("error booking tickets: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("error booking tickets: %s", resp.Status())
	}

	return nil
}
