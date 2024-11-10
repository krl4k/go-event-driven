package domain

// Interfaces for domain events

type ReceiptIssuePublisher interface {
	PublishIssueReceipt(ticket Ticket) error
}

type AppendToTrackerPublisher interface {
	PublishAppendToTracker(ticket Ticket) error
}
