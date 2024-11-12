package domain

type IssueReceiptEvent struct {
	TicketId string `json:"ticket_id"`
	Price    Money  `json:"price"`
}

type AppendToTrackerEvent struct {
	TicketId      string `json:"ticket_id"`
	CustomerEmail string `json:"customer_email"`
	Price         Money  `json:"price"`
}

// Interfaces for domain events

type ReceiptIssuePublisher interface {
	PublishIssueReceipt(event IssueReceiptEvent) error
}

type AppendToTrackerPublisher interface {
	PublishAppendToTracker(event AppendToTrackerEvent) error
}
