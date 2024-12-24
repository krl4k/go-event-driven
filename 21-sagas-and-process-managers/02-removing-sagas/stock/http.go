package stock

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func mountHttpHandlers(e *echo.Echo, repo *Repo) *echo.Route {
	return e.POST("/products-stock", func(c echo.Context) error {
		productStock := ProductStock{}
		if err := c.Bind(&productStock); err != nil {
			return err
		}
		if productStock.Quantity <= 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "quantity must be greater than 0")
		}
		if productStock.ProductID == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "product_id must be provided")
		}

		err := repo.InsertProductStock(c.Request().Context(), InsertProductStockReq{
			ProductID: productStock.ProductID,
			Quantity:  productStock.Quantity,
		})
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusCreated)
	})
}
