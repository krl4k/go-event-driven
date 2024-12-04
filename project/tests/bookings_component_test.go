package tests

import (
	"bytes"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"time"
)

func (suite *ComponentTestSuite) TestBookTickets() {
	showID := uuid.New()
	_, err := suite.db.ExecContext(suite.ctx, `
       INSERT INTO shows (
           id, 
           dead_nation_id,
           number_of_tickets,
           start_time,
           title,
           venue
       ) VALUES (
           $1, $2, $3, $4, $5, $6
       )`,
		showID,
		uuid.New(),
		100,
		time.Now(),
		"Test Show",
		"Test Venue",
	)
	require.NoError(suite.T(), err)

	request := struct {
		ShowID          uuid.UUID `json:"show_id"`
		NumberOfTickets int       `json:"number_of_tickets"`
		CustomerEmail   string    `json:"customer_email"`
	}{
		ShowID:          showID,
		NumberOfTickets: 3,
		CustomerEmail:   "email@example.com",
	}

	payload, err := json.Marshal(request)
	require.NoError(suite.T(), err)

	httpReq, err := http.NewRequest(
		http.MethodPost,
		"http://localhost:8080/book-tickets",
		bytes.NewBuffer(payload),
	)
	require.NoError(suite.T(), err)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := suite.httpClient.Do(httpReq)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var response struct {
		BookingID uuid.UUID `json:"booking_id"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)
	require.NotEqual(suite.T(), uuid.Nil, response.BookingID)

	// check the database record
	var booking struct {
		ID              uuid.UUID `db:"id"`
		ShowID          uuid.UUID `db:"show_id"`
		NumberOfTickets int       `db:"number_of_tickets"`
		CustomerEmail   string    `db:"customer_email"`
	}

	err = suite.db.GetContext(suite.ctx, &booking, `
       SELECT * FROM bookings WHERE id = $1
   `, response.BookingID)
	require.NoError(suite.T(), err)

	// Проверяем значения полей
	assert.Equal(suite.T(), showID, booking.ShowID)
	assert.Equal(suite.T(), 3, booking.NumberOfTickets)
	assert.Equal(suite.T(), "email@example.com", booking.CustomerEmail)
}
