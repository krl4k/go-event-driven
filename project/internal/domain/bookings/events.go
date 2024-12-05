package bookings

type BookingMade struct {
	BookingID       string `json:"booking_id"`
	NumberOfTickets int    `json:"number_of_tickets"`
	CustomerEmail   string `json:"customer_email"`
	ShowID          string `json:"show_id"`
}
