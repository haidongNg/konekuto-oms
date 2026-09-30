package http

import (
	"net/http"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/haidongNg/konekuto-oms/internal/domain"
	"github.com/haidongNg/konekuto-oms/pkg/middlewares"
	"github.com/haidongNg/konekuto-oms/pkg/response"
)

type OrderHandler interface {
	RegisterRoutes(app *fiber.App, jwtSecret string, checker middlewares.BlacklistChecker)
}

type orderHandlerImpl struct {
	useCase domain.OrderUseCase
}

func NewOrderHandler(uc domain.OrderUseCase) OrderHandler {
	return &orderHandlerImpl{useCase: uc}
}

func (h *orderHandlerImpl) RegisterRoutes(app *fiber.App, jwtSecret string, checker middlewares.BlacklistChecker) {
	jwtConfig := jwtware.Config{
		SigningKey: jwtware.SigningKey{Key: []byte(jwtSecret)},
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return response.Error(c, http.StatusUnauthorized, "Token không hợp lệ hoặc đã hết hạn")
		},
	}

	// API Mua hàng phải được bảo vệ
	protectedGroup := app.Group("/api/v1/orders",
		jwtware.New(jwtConfig),
		middlewares.CheckBlacklist(checker), // Ngăn token đã bị logout
	)

	// Fiber v3 sử dụng chữ P viết hoa cho phương thức Post
	protectedGroup.Post("", h.createOrder)
}

func (h *orderHandlerImpl) createOrder(c fiber.Ctx) error {
	// 1. Lấy userID từ Token (hàm ExtractUserClaims cần nhận fiber.Ctx)
	userID, _, _, _, err := middlewares.ExtractUserClaims(c)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, err.Error())
	}

	// 2. Parse dữ liệu Giỏ hàng Client gửi lên (Dùng BodyParser thay vì Bind)
	var req domain.OrderCreateReq
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Dữ liệu JSON không hợp lệ")
	}

	// 3. Xử lý logic (Fiber dùng c.UserContext() thay cho c.Request().Context())
	order, err := h.useCase.CreateOrder(c.Context(), userID, &req)
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "Lỗi khi tạo đơn hàng: "+err.Error())
	}

	return response.Success(c, http.StatusCreated, "Tạo đơn hàng thành công", order)
}
