package repository

import (
	"context"
	"errors"

	"github.com/haidongNg/konekuto-oms/internal/domain"
	"github.com/jmoiron/sqlx"
)

type sqliteOrderRepository struct {
	db *sqlx.DB
}

func NewSQLiteOrderRepository(db *sqlx.DB) domain.OrderRepository {
	return &sqliteOrderRepository{db: db}
}

func (r *sqliteOrderRepository) CreateWithItems(ctx context.Context, order *domain.Order, items []domain.OrderItem) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// 1. Insert Order tổng (Bổ sung order_type và delivery_date)
	orderQuery := `
		INSERT INTO orders (id, user_id, total_amount, status, order_type, payment_method, shipping_address, delivery_date, note, created_at, updated_at)
		VALUES (:id, :user_id, :total_amount, :status, :order_type, :payment_method, :shipping_address, :delivery_date, :note, :created_at, :updated_at)
	`
	if _, err = tx.NamedExecContext(ctx, orderQuery, order); err != nil {
		return err
	}

	// 2. Insert Order Items & Trừ tồn kho nông sản an toàn
	itemQuery := `
		INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, sub_total, created_at)
		VALUES (:id, :order_id, :product_id, :quantity, :unit_price, :sub_total, :created_at)
	`

	stockQuery := `
		UPDATE products 
		SET stock_quantity = stock_quantity - :quantity 
		WHERE id = :product_id AND stock_quantity >= :quantity AND deleted_at IS NULL
	`

	for _, item := range items {
		if _, err = tx.NamedExecContext(ctx, itemQuery, item); err != nil {
			return err
		}

		res, err := tx.NamedExecContext(ctx, stockQuery, item)
		if err != nil {
			return err
		}

		rows, _ := res.RowsAffected()
		if rows == 0 {
			return errors.New("nông sản không đủ sản lượng hoặc đã hết hàng")
		}
	}

	return nil
}
