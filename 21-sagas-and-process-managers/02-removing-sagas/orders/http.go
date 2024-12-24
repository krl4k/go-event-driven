package orders

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"remove_sagas/common"
	"remove_sagas/stock"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

type PostOrderRequest struct {
	OrderID   uuid.UUID         `json:"order_id"`
	Products  map[uuid.UUID]int `json:"products"`
	Shipped   bool              `json:"shipped"`
	Cancelled bool              `json:"cancelled"`
}

type GetOrderResponse struct {
	OrderID   uuid.UUID `json:"order_id" db:"order_id"`
	Shipped   bool      `json:"shipped" db:"shipped"`
	Cancelled bool      `json:"cancelled" db:"cancelled"`
}

func mountHttpHandlers(
	e *echo.Echo,
	orderRepo *Repo,
	stockRepo *stock.Repo,
) {

	e.POST("/orders", func(c echo.Context) error {
		order := PostOrderRequest{}
		if err := c.Bind(&order); err != nil {
			return err
		}

		var err error
		for i := 0; i < 10; i++ {
			err = common.UpdateInTx(
				c.Request().Context(),
				orderRepo.DB,
				sql.LevelSerializable,
				func(ctx context.Context, tx *sqlx.Tx) error {
					err := orderRepo.PlaceOrder(ctx, tx, PlaceOrderReq{
						OrderID:  order.OrderID,
						Products: order.Products,
					})
					if err != nil {
						return fmt.Errorf("place order: %w", err)
					}

					err = stockRepo.RemoveProductsFromStock(ctx, tx, stock.RemoveProductsFromStockReq{
						OrderID:  order.OrderID,
						Products: order.Products,
					})
					if err != nil {
						err := orderRepo.CancelOrder(ctx, tx, order.OrderID)
						if err != nil {
							return fmt.Errorf("cancel order: %w", err)
						}
						// todo should be 409 but we need backwards compatibility
						return c.NoContent(http.StatusCreated)
					}

					err = orderRepo.ShipOrder(ctx, tx, order.OrderID)
					if err != nil {
						return fmt.Errorf("ship order: %w", err)
					}
					err = orderRepo.InsertOrderProduct(ctx, tx, OrderProductInsertReq{
						OrderID:  order.OrderID,
						Products: order.Products,
					})
					if err != nil {
						return fmt.Errorf("insert order product: %w", err)
					}
					return nil
				},
			)

			if err == nil {
				break
			}
		}
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusCreated)
	})

	e.GET("/orders/:order_id", func(c echo.Context) error {
		orderID, err := uuid.Parse(c.Param("order_id"))
		if err != nil {
			return err
		}

		order, err := orderRepo.GetOrder(c.Request().Context(), orderID)
		if err != nil {
			return fmt.Errorf("get order: %w", err)
		}

		return c.JSON(http.StatusOK, PostOrderRequest{
			OrderID:   order.OrderID,
			Shipped:   order.Shipped,
			Cancelled: order.Cancelled,
		})
	})
}
