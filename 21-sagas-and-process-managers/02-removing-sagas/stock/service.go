package stock

import (
	"os"

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

	repo := NewRepo(db)
	mountHttpHandlers(e, repo)
	//mountMessageHandlers(db, eventBus, commandProcessor)
}
