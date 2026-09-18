package http

import (
	"net/http"
	"strconv"

	"github.com/haidongNg/konekuto-oms/internal/domain"
	"github.com/haidongNg/konekuto-oms/pkg/middlewares"
	"github.com/haidongNg/konekuto-oms/pkg/response"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
)

// ProductHandler định nghĩa hợp đồng Router
type ProductHandler interface {
	RegisterRoutes(e *echo.Echo, jwtSecret string, checker middlewares.BlacklistChecker)
}

type productHandlerImpl struct {
	useCase domain.ProductUseCase
}

func NewProductHandler(uc domain.ProductUseCase) ProductHandler {
	return &productHandlerImpl{useCase: uc}
}

func (h *productHandlerImpl) RegisterRoutes(e *echo.Echo, jwtSecret string, checker middlewares.BlacklistChecker) {
	// ==========================================
	// 1. API PUBLIC (Khách hàng xem danh sách)
	// ==========================================
	publicGroup := e.Group("/api/v1/products")
	publicGroup.GET("", h.listProducts)
	publicGroup.GET("/:id", h.getProduct)

	// ==========================================
	// 2. API ADMIN (Quản trị viên quản lý)
	// ==========================================
	jwtConfig := echojwt.Config{
		SigningKey: []byte(jwtSecret),
		ErrorHandler: func(c *echo.Context, err error) error {
			return response.Error(c, http.StatusUnauthorized, "Token không hợp lệ")
		},
	}

	adminGroup := e.Group("/api/v1/admin/products",
		echojwt.WithConfig(jwtConfig),
		middlewares.CheckBlacklist(checker), // Check token bị thu hồi chưa
		middlewares.RequireRole("admin"),    // Chỉ cấp quyền cho Admin
	)

	adminGroup.POST("", h.createProduct)
	adminGroup.PUT("/:id", h.updateProduct)
	adminGroup.DELETE("/:id", h.deleteProduct)
}

// --- CÁC HÀM XỬ LÝ ---

func (h *productHandlerImpl) createProduct(c *echo.Context) error {
	var req domain.ProductCreateReq
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Dữ liệu không hợp lệ")
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Sai định dạng: "+err.Error())
	}

	product, err := h.useCase.CreateProduct(c.Request().Context(), &req)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err.Error())
	}
	return response.Success(c, http.StatusCreated, "Tạo sản phẩm thành công", product)
}

func (h *productHandlerImpl) getProduct(c *echo.Context) error {
	id := c.Param("id")
	product, err := h.useCase.GetProduct(c.Request().Context(), id)
	if err != nil {
		return response.Error(c, http.StatusNotFound, err.Error())
	}
	return response.Success(c, http.StatusOK, "Thành công", product)
}

func (h *productHandlerImpl) listProducts(c *echo.Context) error {
	// Lấy query params, ví dụ: ?page=1&limit=20
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	products, err := h.useCase.ListProducts(c.Request().Context(), page, limit)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Lỗi lấy danh sách sản phẩm")
	}
	return response.Success(c, http.StatusOK, "Thành công", products)
}

func (h *productHandlerImpl) updateProduct(c *echo.Context) error {
	id := c.Param("id")
	var req domain.ProductUpdateReq
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Dữ liệu không hợp lệ")
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Sai định dạng: "+err.Error())
	}

	product, err := h.useCase.UpdateProduct(c.Request().Context(), id, &req)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}
	return response.Success(c, http.StatusOK, "Cập nhật thành công", product)
}

func (h *productHandlerImpl) deleteProduct(c *echo.Context) error {
	id := c.Param("id")
	if err := h.useCase.DeleteProduct(c.Request().Context(), id); err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}
	return response.Success(c, http.StatusOK, "Xóa sản phẩm thành công (Soft Delete)", nil)
}
