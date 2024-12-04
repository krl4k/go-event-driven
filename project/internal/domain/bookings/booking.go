package bookings

type Booking struct {
	Id              string `json:"id"`
	ShowId          string `json:"show_id"`
	NumberOfTickets int    `json:"number_of_tickets"`
	CustomerEmail   string `json:"customer_email"`
}
