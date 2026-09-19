package domain

import (
	"context"
	"time"
)

type Product struct {
	ID            string     `json:"id" db:"id"`
	Name          string     `json:"name" db:"name"`
	Description   string     `json:"description" db:"description"`
	Price         float64    `json:"price" db:"price"`
	Unit          string     `json:"unit" db:"unit"` // MỚI: Đơn vị tính (mớ, kg, túi 500g, con)
	StockQuantity int        `json:"stock_quantity" db:"stock_quantity"`
	Category      string     `json:"category" db:"category"`
	ImageURL      string     `json:"image_url" db:"image_url"`
	Status        string     `json:"status" db:"status"` // active, out_of_season
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt     *time.Time `json:"-" db:"deleted_at"`
}

type ProductCreateReq struct {
	Name          string  `json:"name" validate:"required"`
	Description   string  `json:"description" validate:"omitempty"`
	Price         float64 `json:"price" validate:"required,min=0"`
	Unit          string  `json:"unit" validate:"required"` // MỚI: Bắt buộc điền đơn vị
	StockQuantity int     `json:"stock_quantity" validate:"required,min=0"`
	Category      string  `json:"category" validate:"omitempty"`
	ImageURL      string  `json:"image_url" validate:"omitempty,url"`
}

type ProductUpdateReq struct {
	Name          string  `json:"name" validate:"omitempty"`
	Description   string  `json:"description" validate:"omitempty"`
	Price         float64 `json:"price" validate:"omitempty,min=0"`
	Unit          string  `json:"unit" validate:"omitempty"`
	StockQuantity int     `json:"stock_quantity" validate:"omitempty,min=0"`
	Category      string  `json:"category" validate:"omitempty"`
	ImageURL      string  `json:"image_url" validate:"omitempty,url"`
	Status        string  `json:"status" validate:"omitempty,oneof=active out_of_season"`
}

type ProductRepository interface {
	Create(ctx context.Context, p *Product) error
	GetByID(ctx context.Context, id string) (*Product, error)
	List(ctx context.Context, limit, offset int) ([]Product, error)
	Update(ctx context.Context, p *Product) error
	Delete(ctx context.Context, id string, deletedAt time.Time) error
}

type ProductUseCase interface {
	CreateProduct(ctx context.Context, req *ProductCreateReq) (*Product, error)
	GetProduct(ctx context.Context, id string) (*Product, error)
	ListProducts(ctx context.Context, page, limit int) ([]Product, error)
	UpdateProduct(ctx context.Context, id string, req *ProductUpdateReq) (*Product, error)
	DeleteProduct(ctx context.Context, id string) error
}
