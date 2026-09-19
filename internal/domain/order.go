package domain

import (
	"context"
	"time"
)

type Order struct {
	ID            string    `json:"id" db:"id"`
	UserID        string    `json:"user_id" db:"user_id"`
	TotalAmount   float64   `json:"total_amount" db:"total_amount"`
	Status        string    `json:"status" db:"status"`                     // pending, confirmed, shipping, completed, cancelled
	OrderType     string    `json:"order_type" db:"order_type"`             // pickup, delivery
	PaymentMethod string    `json:"payment_method" db:"payment_method"`     // cod, banking, momo
	ShippingAddr  *string   `json:"shipping_address" db:"shipping_address"` // Cho phép null nếu pickup
	DeliveryDate  string    `json:"delivery_date" db:"delivery_date"`       // MỚI: Ngày gom đơn giao (VD: "2026-09-25")
	Note          string    `json:"note" db:"note"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`

	Items []OrderItem `json:"items" db:"-"`
}

type OrderItem struct {
	ID        string    `json:"id" db:"id"`
	OrderID   string    `json:"order_id" db:"order_id"`
	ProductID string    `json:"product_id" db:"product_id"`
	Quantity  int       `json:"quantity" db:"quantity"`
	UnitPrice float64   `json:"unit_price" db:"unit_price"`
	SubTotal  float64   `json:"sub_total" db:"sub_total"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type OrderCreateReq struct {
	OrderType     string         `json:"order_type" validate:"required,oneof=pickup delivery"`
	PaymentMethod string         `json:"payment_method" validate:"required,oneof=cod banking momo"`
	ShippingAddr  string         `json:"shipping_address" validate:"required_if=OrderType delivery"` // Bắt buộc nếu là delivery
	DeliveryDate  string         `json:"delivery_date" validate:"required"`                          // MỚI: Ngày khách muốn nhận hàng
	Note          string         `json:"note" validate:"omitempty"`
	Items         []OrderItemReq `json:"items" validate:"required,min=1,dive"`
}

type OrderItemReq struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required,min=1"`
}

type OrderRepository interface {
	CreateWithItems(ctx context.Context, order *Order, items []OrderItem) error
}

type OrderUseCase interface {
	CreateOrder(ctx context.Context, userID string, req *OrderCreateReq) (*Order, error)
}
