package http

import (
	"net/http"
	"strconv"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/haidongNg/konekuto-oms/internal/domain"
	"github.com/haidongNg/konekuto-oms/pkg/middlewares"
	"github.com/haidongNg/konekuto-oms/pkg/response"
)

// ProductHandler định nghĩa hợp đồng Router
type ProductHandler interface {
	RegisterRoutes(app *fiber.App, jwtSecret string, checker middlewares.BlacklistChecker)
}

type productHandlerImpl struct {
	useCase domain.ProductUseCase
}

func NewProductHandler(uc domain.ProductUseCase) ProductHandler {
	return &productHandlerImpl{useCase: uc}
}

func (h *productHandlerImpl) RegisterRoutes(app *fiber.App, jwtSecret string, checker middlewares.BlacklistChecker) {
	// ==========================================
	// 1. API PUBLIC (Khách hàng xem danh sách)
	// ==========================================
	publicGroup := app.Group("/api/v1/products")
	publicGroup.Get("", h.listProducts)
	publicGroup.Get("/:id", h.getProduct)

	// ==========================================
	// 2. API ADMIN (Quản trị viên quản lý)
	// ==========================================
	jwtConfig := jwtware.Config{
		SigningKey: jwtware.SigningKey{Key: []byte(jwtSecret)},
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return response.Error(c, http.StatusUnauthorized, "Token không hợp lệ")
		},
	}

	adminGroup := app.Group("/api/v1/admin/products",
		jwtware.New(jwtConfig),
		middlewares.CheckBlacklist(checker), // Check token bị thu hồi chưa
		middlewares.RequireRole("admin"),    // Chỉ cấp quyền cho Admin
	)

	adminGroup.Post("", h.createProduct)
	adminGroup.Put("/:id", h.updateProduct)
	adminGroup.Delete("/:id", h.deleteProduct)
}

// --- CÁC HÀM XỬ LÝ ---

func (h *productHandlerImpl) createProduct(c fiber.Ctx) error {
	var req domain.ProductCreateReq
	// Fiber v3 dùng c.Bind().Body() để parse và tự động validate luôn
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Dữ liệu không hợp lệ hoặc sai định dạng: "+err.Error())
	}

	product, err := h.useCase.CreateProduct(c.Context(), &req)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err.Error())
	}
	return response.Success(c, http.StatusCreated, "Tạo sản phẩm thành công", product)
}

func (h *productHandlerImpl) getProduct(c fiber.Ctx) error {
	// Dùng c.Params thay vì c.Param
	id := c.Params("id")
	product, err := h.useCase.GetProduct(c.Context(), id)
	if err != nil {
		return response.Error(c, http.StatusNotFound, err.Error())
	}
	return response.Success(c, http.StatusOK, "Thành công", product)
}

func (h *productHandlerImpl) listProducts(c fiber.Ctx) error {
	// Lấy query params bằng c.Query thay vì c.QueryParam
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	products, err := h.useCase.ListProducts(c.Context(), page, limit)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Lỗi lấy danh sách sản phẩm")
	}
	return response.Success(c, http.StatusOK, "Thành công", products)
}

func (h *productHandlerImpl) updateProduct(c fiber.Ctx) error {
	id := c.Params("id")
	var req domain.ProductUpdateReq

	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Dữ liệu không hợp lệ hoặc sai định dạng: "+err.Error())
	}

	product, err := h.useCase.UpdateProduct(c.Context(), id, &req)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}
	return response.Success(c, http.StatusOK, "Cập nhật thành công", product)
}

func (h *productHandlerImpl) deleteProduct(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.useCase.DeleteProduct(c.Context(), id); err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}
	return response.Success(c, http.StatusOK, "Xóa sản phẩm thành công (Soft Delete)", nil)
}
