package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/haidongNg/konekuto-oms/internal/domain"
)

type orderUseCase struct {
	orderRepo      domain.OrderRepository
	productRepo    domain.ProductRepository
	contextTimeout time.Duration
}

func NewOrderUseCase(or domain.OrderRepository, pr domain.ProductRepository, timeout time.Duration) domain.OrderUseCase {
	return &orderUseCase{
		orderRepo:      or,
		productRepo:    pr,
		contextTimeout: timeout,
	}
}

func (u *orderUseCase) CreateOrder(c context.Context, userID string, req *domain.OrderCreateReq) (*domain.Order, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	now := time.Now()
	orderID := uuid.New().String()
	var totalAmount float64
	var orderItems []domain.OrderItem

	// 1. Kiểm tra từng nông sản trong danh sách
	for _, itemReq := range req.Items {
		product, err := u.productRepo.GetByID(ctx, itemReq.ProductID)
		if err != nil {
			return nil, err
		}
		if product == nil {
			return nil, errors.New("sản phẩm không tồn tại: " + itemReq.ProductID)
		}

		if product.Status == "out_of_season" {
			return nil, errors.New("sản phẩm hiện đang hết mùa thu hoạch: " + product.Name)
		}

		if product.StockQuantity < itemReq.Quantity {
			return nil, errors.New("sản lượng hiện tại không đủ đáp ứng: " + product.Name)
		}

		subTotal := product.Price * float64(itemReq.Quantity)
		totalAmount += subTotal

		orderItems = append(orderItems, domain.OrderItem{
			ID:        uuid.New().String(),
			OrderID:   orderID,
			ProductID: itemReq.ProductID,
			Quantity:  itemReq.Quantity,
			UnitPrice: product.Price,
			SubTotal:  subTotal,
			CreatedAt: now,
		})
	}

	// 2. Xử lý địa chỉ nhận hàng theo hình thức
	var shippingAddr *string
	if req.OrderType == "delivery" {
		addr := req.ShippingAddr
		shippingAddr = &addr
	}

	// 3. Khởi tạo Entity Order hoàn chỉnh
	order := &domain.Order{
		ID:            orderID,
		UserID:        userID,
		TotalAmount:   totalAmount,
		Status:        "pending",
		OrderType:     req.OrderType,
		PaymentMethod: req.PaymentMethod,
		ShippingAddr:  shippingAddr,
		DeliveryDate:  req.DeliveryDate,
		Note:          req.Note,
		CreatedAt:     now,
		UpdatedAt:     now,
		Items:         orderItems,
	}

	// 4. Lưu đơn và trừ kho an toàn qua Database Transaction
	if err := u.orderRepo.CreateWithItems(ctx, order, orderItems); err != nil {
		return nil, err
	}

	return order, nil
}
