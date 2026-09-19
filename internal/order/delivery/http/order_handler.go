package http

import (
	"net/http"

	"github.com/haidongNg/konekuto-oms/internal/domain"
	"github.com/haidongNg/konekuto-oms/pkg/middlewares"
	"github.com/haidongNg/konekuto-oms/pkg/response"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
)

type OrderHandler interface {
	RegisterRoutes(e *echo.Echo, jwtSecret string, checker middlewares.BlacklistChecker)
}

type orderHandlerImpl struct {
	useCase domain.OrderUseCase
}

func NewOrderHandler(uc domain.OrderUseCase) OrderHandler {
	return &orderHandlerImpl{useCase: uc}
}

func (h *orderHandlerImpl) RegisterRoutes(e *echo.Echo, jwtSecret string, checker middlewares.BlacklistChecker) {
	jwtConfig := echojwt.Config{
		SigningKey: []byte(jwtSecret),
		ErrorHandler: func(c *echo.Context, err error) error {
			return response.Error(c, http.StatusUnauthorized, "Token không hợp lệ hoặc đã hết hạn")
		},
	}

	// API Mua hàng phải được bảo vệ
	protectedGroup := e.Group("/api/v1/orders",
		echojwt.WithConfig(jwtConfig),
		middlewares.CheckBlacklist(checker), // Ngăn token đã bị logout
	)

	protectedGroup.POST("", h.createOrder)
}

func (h *orderHandlerImpl) createOrder(c *echo.Context) error {
	// 1. Lấy userID từ Token
	userID, _, _, _, err := middlewares.ExtractUserClaims(c)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, err.Error())
	}

	// 2. Parse dữ liệu Giỏ hàng Client gửi lên
	var req domain.OrderCreateReq
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Dữ liệu JSON không hợp lệ")
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Sai định dạng: "+err.Error())
	}

	// 3. Xử lý logic
	order, err := h.useCase.CreateOrder(c.Request().Context(), userID, &req)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "Lỗi khi tạo đơn hàng: "+err.Error())
	}

	return response.Success(c, http.StatusCreated, "Tạo đơn hàng thành công", order)
}
