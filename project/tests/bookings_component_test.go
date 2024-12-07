package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"sync"
	"sync/atomic"
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
	calls := atomic.Int32{}
	suite.deadNationMock.EXPECT().BookTickets(gomock.Any(), gomock.Any()).
		Return(nil).
		Do(func(arg0, arg1 interface{}) {
			calls.Add(1)
		}).
		Times(1)

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

	require.Eventually(
		suite.T(),
		func() bool {
			return calls.Load() == 1
		},
		15*time.Second,
		100*time.Millisecond,
		"All mocks should have been called",
	)
}

func (suite *ComponentTestSuite) TestBookTicketsOverbooking() {
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
		NumberOfTickets: 101,
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
	require.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
}

func (suite *ComponentTestSuite) TestBookTicketsConcurrent() {
	// Создаем шоу с ограниченным количеством билетов
	showID := uuid.New()
	totalTickets := 10
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
		totalTickets,
		time.Now(),
		"Test Show",
		"Test Venue",
	)
	require.NoError(suite.T(), err)

	// Количество конкурентных запросов
	numRequests := 5
	ticketsPerRequest := 3 // 5 * 3 = 15 билетов (больше чем доступно)

	// Канал для сбора результатов
	results := make(chan int, numRequests)
	calls := atomic.Int32{}
	suite.deadNationMock.EXPECT().BookTickets(gomock.Any(), gomock.Any()).
		Return(nil).
		Do(func(arg0, arg1 interface{}) {
			calls.Add(1)
		}).
		Times(3)
	// Создаем WaitGroup для синхронизации горутин
	var wg sync.WaitGroup
	wg.Add(numRequests)

	// Запускаем конкурентные запросы
	for i := 0; i < numRequests; i++ {
		go func(index int) {
			defer wg.Done()

			request := struct {
				ShowID          uuid.UUID `json:"show_id"`
				NumberOfTickets int       `json:"number_of_tickets"`
				CustomerEmail   string    `json:"customer_email"`
			}{
				ShowID:          showID,
				NumberOfTickets: ticketsPerRequest,
				CustomerEmail:   fmt.Sprintf("email%d@example.com", index),
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

			results <- resp.StatusCode
		}(i)
	}

	// Ждем завершения всех запросов
	wg.Wait()
	close(results)

	// Подсчитываем успешные и неуспешные бронирования
	successCount := 0
	failureCount := 0

	for status := range results {
		if status == http.StatusCreated {
			successCount++
		} else if status == http.StatusBadRequest {
			failureCount++
		}
	}

	fmt.Println("successCount", successCount)
	fmt.Println("failureCount", failureCount)

	// Проверяем, что общее количество забронированных билетов не превышает доступное
	var totalBooked int
	err = suite.db.GetContext(suite.ctx, &totalBooked, `
        SELECT COALESCE(SUM(number_of_tickets), 0) 
        FROM bookings 
        WHERE show_id = $1
    `, showID)
	require.NoError(suite.T(), err)

	// Проверяем условия
	assert.LessOrEqual(suite.T(), totalBooked, totalTickets, "Total booked tickets should not exceed available tickets")
	assert.True(suite.T(), successCount > 0, "At least one booking should succeed")
	assert.True(suite.T(), failureCount > 0, "Some bookings should fail due to overbooking")
	assert.Equal(suite.T(), numRequests, successCount+failureCount, "All requests should either succeed or fail")

	require.Eventually(
		suite.T(),
		func() bool {
			return calls.Load() == 3
		},
		15*time.Second,
		100*time.Millisecond,
		"All mocks should have been called",
	)
}
