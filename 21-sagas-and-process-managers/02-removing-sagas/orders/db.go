package orders

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func initializeDatabaseSchema(db *sqlx.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS orders (
			order_id UUID PRIMARY KEY,
			shipped BOOLEAN NOT NULL,
			cancelled BOOLEAN NOT NULL
		);

		CREATE TABLE IF NOT EXISTS order_products (
			order_id UUID NOT NULL,
			product_id UUID NOT NULL,
			quantity INT NOT NULL,

		    PRIMARY KEY (order_id, product_id)
		);
	`)
	if err != nil {
		panic(err)
	}
}

type Repo struct {
	DB *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{DB: db}
}

type PlaceOrderReq struct {
	OrderID  uuid.UUID         `json:"order_id"`
	Products map[uuid.UUID]int `json:"products"`
}

func (r *Repo) PlaceOrder(ctx context.Context, tx *sqlx.Tx, req PlaceOrderReq) error {
	_, err := tx.Exec(
		"INSERT INTO orders (order_id, shipped, cancelled) VALUES ($1, $2, $3)",
		req.OrderID,
		false,
		false,
	)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	return nil
}

type OrderProductInsertReq struct {
	OrderID  uuid.UUID
	Products map[uuid.UUID]int
}

func (r *Repo) InsertOrderProduct(ctx context.Context, tx *sqlx.Tx, req OrderProductInsertReq) error {
	for product, quantity := range req.Products {
		_, err := tx.Exec(
			"INSERT INTO order_products (order_id, product_id, quantity) VALUES ($1, $2, $3)",
			req.OrderID,
			product,
			quantity,
		)
		if err != nil {
			return fmt.Errorf("insert order product: %w", err)
		}
	}

	return nil
}

func (r *Repo) GetOrder(ctx context.Context, orderID uuid.UUID) (*GetOrderResponse, error) {
	order := &GetOrderResponse{}
	err := r.DB.Get(order,
		"SELECT order_id, shipped, cancelled FROM orders WHERE order_id = $1",
		orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("select order: %w", err)
	}

	return order, nil
}

func (r *Repo) CancelOrder(ctx context.Context, tx *sqlx.Tx, orderID uuid.UUID) error {
	_, err := tx.Exec("UPDATE orders SET cancelled = true WHERE order_id = $1", orderID)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	return nil
}

func (r *Repo) ShipOrder(ctx context.Context, tx *sqlx.Tx, orderID uuid.UUID) error {
	_, err := tx.Exec("UPDATE orders SET shipped = true WHERE order_id = $1", orderID)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	return nil
}
