package domain

import "context"

type ReceiptsIssuer interface {
	IssueReceipt(ctx context.Context, ticketID Ticket) error
}

type SpreadsheetsAppender interface {
	AppendToTracker(ctx context.Context, ticketID Ticket) error
}
