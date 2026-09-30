package http

import (
	"net/http"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/haidongNg/konekuto-oms/internal/domain"
	"github.com/haidongNg/konekuto-oms/pkg/middlewares"
	"github.com/haidongNg/konekuto-oms/pkg/response"
)

// UserHandler định nghĩa giao thức cho HTTP Delivery
type UserHandler interface {
	RegisterRoutes(app *fiber.App, jwtSecret string)
}

// userHandlerImpl là bản thực thi (Private)
type userHandlerImpl struct {
	useCase domain.UserUseCase
}

// NewUserHandler trả về Interface UserHandler
func NewUserHandler(us domain.UserUseCase) UserHandler {
	return &userHandlerImpl{
		useCase: us,
	}
}

// RegisterRoutes gắn các API endpoint vào Fiber Router
func (h *userHandlerImpl) RegisterRoutes(app *fiber.App, jwtSecret string) {
	// ==========================================
	// 1. API PUBLIC (Không cần Token)
	// ==========================================
	publicGroup := app.Group("/api/v1/users")
	publicGroup.Post("/register", h.register)
	publicGroup.Post("/login", h.login)
	publicGroup.Post("/refresh-token", h.refreshToken) // Cấp lại token mới bằng Refresh Token

	// Cấu hình Middleware phân giải JWT cho Fiber v3
	jwtConfig := jwtware.Config{
		SigningKey: jwtware.SigningKey{Key: []byte(jwtSecret)},
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return response.Error(c, http.StatusUnauthorized, "Token không hợp lệ hoặc đã hết hạn")
		},
	}

	// ==========================================
	// 2. API PRIVATE (Yêu cầu đăng nhập)
	// ==========================================
	protectedGroup := app.Group("/api/v1/users/me",
		jwtware.New(jwtConfig),
		middlewares.CheckBlacklist(h.useCase),
	)
	protectedGroup.Get("", h.getProfile)
	protectedGroup.Post("/logout", h.logout)

	// ==========================================
	// 3. API ADMIN (Yêu cầu quyền Quản trị)
	// ==========================================
	// adminGroup := app.Group("/api/v1/admin/users",
	// 	jwtware.New(jwtConfig),
	// 	middlewares.CheckBlacklist(h.useCase),
	// 	middlewares.RequireRole("admin"),
	// )
	// adminGroup.Get("", h.adminGetUsers)
}

// --- CÁC HÀM XỬ LÝ (CONTROLLERS) ---

func (h *userHandlerImpl) register(c fiber.Ctx) error {
	var req domain.UserRegisterReq
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Dữ liệu JSON không hợp lệ hoặc sai định dạng: "+err.Error())
	}

	user, err := h.useCase.Register(c.Context(), &req)
	if err != nil {
		return response.Error(c, http.StatusConflict, err.Error())
	}
	return response.Success(c, http.StatusCreated, "Đăng ký thành công", user)
}

func (h *userHandlerImpl) login(c fiber.Ctx) error {
	var req domain.UserLoginReq
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Dữ liệu JSON không hợp lệ hoặc sai định dạng: "+err.Error())
	}

	res, err := h.useCase.Login(c.Context(), &req)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, err.Error())
	}
	return response.Success(c, http.StatusOK, "Đăng nhập thành công", res)
}

func (h *userHandlerImpl) refreshToken(c fiber.Ctx) error {
	var req domain.RefreshTokenReq
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Vui lòng cung cấp refresh_token hợp lệ: "+err.Error())
	}

	res, err := h.useCase.RefreshToken(c.Context(), &req)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, err.Error())
	}
	return response.Success(c, http.StatusOK, "Cấp lại token thành công", res)
}

func (h *userHandlerImpl) logout(c fiber.Ctx) error {
	// 1. Lấy JTI và thời gian hết hạn từ Header thông qua helper
	_, _, jti, exp, err := middlewares.ExtractUserClaims(c)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, "Token không hợp lệ")
	}

	// 2. Client gửi kèm Refresh Token dưới dạng Body JSON
	var req domain.RefreshTokenReq
	_ = c.Bind().Body(&req) // Bỏ qua bắt lỗi nếu client không gửi body

	// 3. Đưa Access Token vào danh sách đen & Xóa Refresh Token
	err = h.useCase.Logout(c.Context(), jti, req.RefreshToken, exp)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Lỗi khi đăng xuất: "+err.Error())
	}

	return response.Success(c, http.StatusOK, "Đăng xuất thành công", nil)
}

func (h *userHandlerImpl) getProfile(c fiber.Ctx) error {
	// Lấy dữ liệu user từ claims thông qua helper
	userID, role, _, _, err := middlewares.ExtractUserClaims(c)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, err.Error())
	}

	return response.Success(c, http.StatusOK, "Lấy thông tin profile thành công", map[string]any{
		"user_id": userID,
		"role":    role,
	})
}
