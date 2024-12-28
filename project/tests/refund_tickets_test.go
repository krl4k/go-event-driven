package tests

//
//import (
//	"bytes"
//	"encoding/json"
//	"fmt"
//	"github.com/golang/mock/gomock"
//	"github.com/google/uuid"
//	"github.com/stretchr/testify/require"
//	"net/http"
//	"sync/atomic"
//	"time"
//)
//
//func (suite *ComponentTestSuite) TestRefundTicket() {
//	// Setup - first create a show and book a ticket to get a valid ticket BookingID
//	showID := uuid.New()
//	_, err := suite.db.ExecContext(suite.ctx, `
//		INSERT INTO shows (
//			id,
//			dead_nation_id,
//			number_of_tickets,
//			start_time,
//			title,
//			venue
//		) VALUES (
//			$1, $2, $3, $4, $5, $6
//		)`,
//		showID,
//		uuid.New(),
//		100,
//		time.Now(),
//		"Test Show",
//		"Test Venue",
//	)
//	require.NoError(suite.T(), err)
//
//	calls := atomic.Int32{}
//
//	suite.deadNationMock.EXPECT().BookTickets(gomock.Any(), gomock.Any()).
//		Return(nil).
//		Do(func(arg0, arg1 interface{}) {
//			calls.Add(1)
//		}).Times(1)
//
//	suite.paymentsMock.EXPECT().
//		Refund(gomock.Any(), gomock.Any(), gomock.Any()).
//		Return(nil).
//		Do(func(arg0, arg1, arg2 interface{}) {
//			calls.Add(1)
//		}).
//		Times(1)
//
//	suite.receiptsMock.EXPECT().
//		VoidReceipt(gomock.Any(), gomock.Any(), gomock.Any()).
//		Return(nil).
//		Do(func(arg0, arg1, arg2 interface{}) {
//			calls.Add(1)
//		}).
//		Times(1)
//
//	// Book a ticket first to get a valid ticket BookingID
//	bookRequest := struct {
//		ShowID          uuid.UUID `json:"show_id"`
//		NumberOfTickets int       `json:"number_of_tickets"`
//		CustomerEmail   string    `json:"customer_email"`
//	}{
//		ShowID:          showID,
//		NumberOfTickets: 1,
//		CustomerEmail:   "test@example.com",
//	}
//
//	bookPayload, err := json.Marshal(bookRequest)
//	require.NoError(suite.T(), err)
//
//	bookHttpReq, err := http.NewRequest(
//		http.MethodPost,
//		"http://localhost:8080/book-tickets",
//		bytes.NewBuffer(bookPayload),
//	)
//	require.NoError(suite.T(), err)
//	bookHttpReq.Header.Set("Content-Type", "application/json")
//
//	bookResp, err := suite.httpClient.Do(bookHttpReq)
//	require.NoError(suite.T(), err)
//	require.Equal(suite.T(), http.StatusCreated, bookResp.StatusCode)
//
//	var bookResponse struct {
//		BookingID uuid.UUID `json:"booking_id"`
//	}
//	err = json.NewDecoder(bookResp.Body).Decode(&bookResponse)
//	require.NoError(suite.T(), err)
//
//	refundHttpReq, err := http.NewRequest(
//		http.MethodPut,
//		fmt.Sprintf("http://localhost:8080/ticket-refund/%s", bookResponse.BookingID),
//		nil,
//	)
//	require.NoError(suite.T(), err)
//	refundHttpReq.Header.Set("Content-Type", "application/json")
//
//	// Make the refund request
//	refundResp, err := suite.httpClient.Do(refundHttpReq)
//	require.NoError(suite.T(), err)
//	require.Equal(suite.T(), http.StatusAccepted, refundResp.StatusCode)
//
//	require.Eventually(
//		suite.T(),
//		func() bool {
//			return calls.Load() == 3
//		},
//		10*time.Second,
//		100*time.Millisecond,
//		"All mocks should have been called",
//	)
//}
//
////
////func (suite *ComponentTestSuite) TestRefundTicketIdempotency() {
////	// Similar setup as above
////	showID := uuid.New()
////	bookingID := uuid.New()
////	idempotencyKey := uuid.New().String()
////
////	// Setup show and booking in the database...
////
////	// Make the same refund request twice
////	for i := 0; i < 2; i++ {
////		refundHttpReq, err := http.NewRequest(
////			http.MethodPut,
////			fmt.Sprintf("http://localhost:8080/ticket-refund/%s", bookingID),
////			nil,
////		)
////		require.NoError(suite.T(), err)
////		refundHttpReq.Header.Set("Content-Type", "application/json")
////		refundHttpReq.Header.Set("Idempotency-Key", idempotencyKey)
////
////		refundResp, err := suite.httpClient.Do(refundHttpReq)
////		require.NoError(suite.T(), err)
////		require.Equal(suite.T(), http.StatusOK, refundResp.StatusCode)
////	}
////
////	// Verify that the services were only called once despite two requests
////	mockPaymentService.AssertNumberOfCalls(suite.T(), "Refund", 1)
////	mockReceiptsService.AssertNumberOfCalls(suite.T(), "VoidReceipt", 1)
////}
//
////func (suite *ComponentTestSuite) TestRefundTicketError() {
////	bookingID := uuid.New()
////	idempotencyKey := uuid.New().String()
////
////	// Configure mock to return an error
////	mockPaymentService.On("Refund", mock.Anything, bookingID, idempotencyKey).
////		Return(errors.New("payment service error"))
////
////	refundHttpReq, err := http.NewRequest(
////		http.MethodPut,
////		fmt.Sprintf("http://localhost:8080/ticket-refund/%s", bookingID),
////		nil,
////	)
////	require.NoError(suite.T(), err)
////	refundHttpReq.Header.Set("Content-Type", "application/json")
////	refundHttpReq.Header.Set("Idempotency-Key", idempotencyKey)
////
////	refundResp, err := suite.httpClient.Do(refundHttpReq)
////	require.NoError(suite.T(), err)
////	require.Equal(suite.T(), http.StatusInternalServerError, refundResp.StatusCode)
////
////	// Verify that only payment service was called and receipts service was not
////	mockPaymentService.AssertNumberOfCalls(suite.T(), "Refund", 1)
////	mockReceiptsService.AssertNumberOfCalls(suite.T(), "VoidReceipt", 0)
////}
