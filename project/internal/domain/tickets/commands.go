package domain

import domain2 "tickets/internal/domain"

type RefundTicket struct {
	Header   domain2.EventHeader `json:"header"`
	TicketId string              `json:"ticket_id"`
}
