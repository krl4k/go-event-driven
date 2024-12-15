package entities

type RefundTicket struct {
	Header   EventHeader `json:"header"`
	TicketId string      `json:"ticket_id"`
}
