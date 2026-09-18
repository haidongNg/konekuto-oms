package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/haidongNg/konekuto-oms/internal/domain"
)

// productUseCase là bản thực thi (Private)
type productUseCase struct {
	productRepo    domain.ProductRepository
	contextTimeout time.Duration
}

// NewProductUseCase trả về Interface ProductUseCase
func NewProductUseCase(pr domain.ProductRepository, timeout time.Duration) domain.ProductUseCase {
	return &productUseCase{
		productRepo:    pr,
		contextTimeout: timeout,
	}
}

func (u *productUseCase) CreateProduct(c context.Context, req *domain.ProductCreateReq) (*domain.Product, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	now := time.Now()
	product := &domain.Product{
		ID:            uuid.New().String(),
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		StockQuantity: req.StockQuantity,
		Category:      req.Category,
		ImageURL:      req.ImageURL,
		Status:        "active", // Mặc định khi tạo mới là active
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := u.productRepo.Create(ctx, product); err != nil {
		return nil, err
	}
	return product, nil
}

func (u *productUseCase) GetProduct(c context.Context, id string) (*domain.Product, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	product, err := u.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, errors.New("không tìm thấy sản phẩm")
	}
	return product, nil
}

func (u *productUseCase) ListProducts(c context.Context, page, limit int) ([]domain.Product, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Tính toán Offset dùng cho SQL phân trang
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20 // Mặc định 20 sp/trang
	}
	offset := (page - 1) * limit

	return u.productRepo.List(ctx, limit, offset)
}

func (u *productUseCase) UpdateProduct(c context.Context, id string, req *domain.ProductUpdateReq) (*domain.Product, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Kiểm tra xem sản phẩm có tồn tại không
	existingProduct, err := u.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existingProduct == nil {
		return nil, errors.New("không tìm thấy sản phẩm")
	}

	// Ghi đè các trường có dữ liệu mới (Partial Update)
	if req.Name != "" {
		existingProduct.Name = req.Name
	}
	if req.Description != "" {
		existingProduct.Description = req.Description
	}
	if req.Price > 0 {
		existingProduct.Price = req.Price
	}
	if req.StockQuantity >= 0 {
		existingProduct.StockQuantity = req.StockQuantity
	}
	if req.Category != "" {
		existingProduct.Category = req.Category
	}
	if req.ImageURL != "" {
		existingProduct.ImageURL = req.ImageURL
	}
	if req.Status != "" {
		existingProduct.Status = req.Status
	}
	existingProduct.UpdatedAt = time.Now()

	// Lưu xuống DB
	if err := u.productRepo.Update(ctx, existingProduct); err != nil {
		return nil, err
	}
	return existingProduct, nil
}

func (u *productUseCase) DeleteProduct(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Xóa mềm: truyền time.Now() xuống Repo để cập nhật deleted_at
	return u.productRepo.Delete(ctx, id, time.Now())
}
