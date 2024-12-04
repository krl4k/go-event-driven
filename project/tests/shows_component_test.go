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

func (suite *ComponentTestSuite) TestCreateShow() {
	// Test data
	deadNationID := uuid.MustParse("d0b9d5a0-8e1f-4b1a-9f1a-0e8f5e6b9a1a")
	startTime, err := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
	require.NoError(suite.T(), err)

	request := struct {
		DeadNationID    uuid.UUID `json:"dead_nation_id"`
		NumberOfTickets int       `json:"number_of_tickets"`
		StartTime       time.Time `json:"start_time"`
		Title           string    `json:"title"`
		Venue           string    `json:"venue"`
	}{
		DeadNationID:    deadNationID,
		NumberOfTickets: 100,
		StartTime:       startTime,
		Title:           "The best show ever",
		Venue:           "The best venue ever",
	}

	//suite.receiptsMock.EXPECT().
	//	IssueReceipt(
	//		gomock.Any(),
	//		gomock.Any(),
	//	).
	//	Return(nil, nil).
	//	Times(1)

	//suite.spreadsheetsMock.EXPECT().
	//	AppendRow(
	//		gomock.Any(),
	//		gomock.Any(),
	//	).
	//	Return(nil).
	//	Times(1)

	//suite.filesMock.EXPECT().Upload(
	//	gomock.Any(),
	//	gomock.Any(),
	//	gomock.Any(),
	//).
	//	Return(nil).
	//	Times(1)

	// Создаем HTTP запрос
	payload, err := json.Marshal(request)
	require.NoError(suite.T(), err)

	httpReq, err := http.NewRequest(
		http.MethodPost,
		"http://localhost:8080/shows",
		bytes.NewBuffer(payload),
	)
	require.NoError(suite.T(), err)

	httpReq.Header.Set("Content-Type", "application/json")

	// Отправляем запрос
	resp, err := suite.httpClient.Do(httpReq)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	// Проверяем ответ
	var response struct {
		ShowID uuid.UUID `json:"show_id"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)
	require.NotEqual(suite.T(), uuid.Nil, response.ShowID)

	// Проверяем запись в БД
	var show struct {
		ID              uuid.UUID `db:"id"`
		DeadNationID    uuid.UUID `db:"dead_nation_id"`
		NumberOfTickets int       `db:"number_of_tickets"`
		StartTime       time.Time `db:"start_time"`
		Title           string    `db:"title"`
		Venue           string    `db:"venue"`
	}

	err = suite.db.GetContext(suite.ctx, &show, `
        SELECT * FROM shows WHERE id = $1
    `, response.ShowID)
	require.NoError(suite.T(), err)

	// Проверяем поля
	assert.Equal(suite.T(), deadNationID, show.DeadNationID)
	assert.Equal(suite.T(), 100, show.NumberOfTickets)
	assert.Equal(suite.T(), startTime.UTC(), show.StartTime.UTC())
	assert.Equal(suite.T(), "The best show ever", show.Title)
	assert.Equal(suite.T(), "The best venue ever", show.Venue)
}
