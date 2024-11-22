package domain

type AppendToTrackerRequest struct {
	Rows            []string
	SpreadsheetName string
}
