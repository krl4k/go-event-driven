package domain

type RefundTicket struct {
	Header   EventHeader
	TicketId string `json:"ticket_id"`
}
