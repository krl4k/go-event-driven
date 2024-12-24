package orders

import (
	"os"
	"remove_sagas/stock"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

func Initialize(
	e *echo.Echo,
) {
	db, err := sqlx.Open("postgres", os.Getenv("POSTGRES_URL"))
	if err != nil {
		panic(err)
	}

	initializeDatabaseSchema(db)

	orderRepo := NewRepo(db)
	stockRepo := stock.NewRepo(db)
	mountHttpHandlers(e, orderRepo, stockRepo)
	//mountMessageHandlers(db, commandBus, eventProcessor, commandProcessor)
}
