package stock

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func initializeDatabaseSchema(db *sqlx.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS stock (
			product_id UUID PRIMARY KEY,
			quantity INT NOT NULL
		);
	`)
	if err != nil {
		panic(err)
	}
}

type ProductStock struct {
	ProductID string `db:"product_id" json:"product_id"`
	Quantity  int    `db:"quantity" json:"quantity"`
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

type InsertProductStockReq struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

func (r *Repo) InsertProductStock(ctx context.Context, req InsertProductStockReq) error {
	_, err := r.db.Exec(`
			INSERT INTO stock (product_id, quantity)
			VALUES ($1, $2)
			ON CONFLICT (product_id) DO UPDATE SET quantity = stock.quantity + $2
	`, req.ProductID, req.Quantity)
	return err
}

type RemoveProductsFromStockReq struct {
	OrderID  uuid.UUID         `json:"order_id"`
	Products map[uuid.UUID]int `json:"products"`
}

var ProductsOutOfStockError = fmt.Errorf("products out of stock")

func (r *Repo) RemoveProductsFromStock(ctx context.Context, tx *sqlx.Tx, req RemoveProductsFromStockReq) error {
	missingProducts := make(map[uuid.UUID]int)

	for productID, quantity := range req.Products {
		quantityInStock := 0
		err := tx.Get(&quantityInStock,
			"SELECT quantity FROM stock WHERE product_id = $1",
			productID,
		)
		if err != nil {
			return fmt.Errorf("get quantity in stock: %w", err)
		}

		if quantityInStock < quantity {
			missingProducts[productID] = quantity - quantityInStock
		}

		if len(missingProducts) > 0 {
			continue
		}

		_, err = tx.Exec(
			"UPDATE stock SET quantity = quantity - $1 WHERE product_id = $2",
			quantity,
			productID,
		)
		if err != nil {
			return fmt.Errorf("update stock: %w", err)
		}
	}

	if len(missingProducts) > 0 {
		return ProductsOutOfStockError
	}

	return nil
}
