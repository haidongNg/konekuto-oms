package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/haidongNg/konekuto-oms/internal/domain"
	"github.com/jmoiron/sqlx"
)

type sqliteProductRepository struct {
	db *sqlx.DB
}

func NewSQLiteProductRepository(db *sqlx.DB) domain.ProductRepository {
	return &sqliteProductRepository{db: db}
}

func (r *sqliteProductRepository) Create(ctx context.Context, p *domain.Product) error {
	query := `
		INSERT INTO products (id, name, description, price, unit, stock_quantity, category, image_url, status, created_at, updated_at)
		VALUES (:id, :name, :description, :price, :unit, :stock_quantity, :category, :image_url, :status, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, p)
	return err
}

func (r *sqliteProductRepository) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	var product domain.Product
	query := `SELECT * FROM products WHERE id = ? AND deleted_at IS NULL`

	err := r.db.GetContext(ctx, &product, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

func (r *sqliteProductRepository) List(ctx context.Context, limit, offset int) ([]domain.Product, error) {
	var products []domain.Product
	query := `
		SELECT * FROM products 
		WHERE deleted_at IS NULL 
		ORDER BY created_at DESC 
		LIMIT ? OFFSET ?
	`
	err := r.db.SelectContext(ctx, &products, query, limit, offset)
	if err != nil {
		return nil, err
	}
	if products == nil {
		products = []domain.Product{}
	}
	return products, nil
}

func (r *sqliteProductRepository) Update(ctx context.Context, p *domain.Product) error {
	query := `
		UPDATE products 
		SET name = :name, description = :description, price = :price, unit = :unit,
		    stock_quantity = :stock_quantity, category = :category, 
		    image_url = :image_url, status = :status, updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL
	`
	_, err := r.db.NamedExecContext(ctx, query, p)
	return err
}

func (r *sqliteProductRepository) Delete(ctx context.Context, id string, deletedAt time.Time) error {
	query := `UPDATE products SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`
	res, err := r.db.ExecContext(ctx, query, deletedAt, id)
	if err != nil {
		return err
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("không tìm thấy sản phẩm hoặc sản phẩm đã bị xóa")
	}
	return nil
}
