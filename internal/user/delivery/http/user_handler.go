package http

import (
	"net/http"

	"github.com/haidongNg/konekuto-oms/internal/domain"
	"github.com/haidongNg/konekuto-oms/pkg/middlewares"
	"github.com/haidongNg/konekuto-oms/pkg/response"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
)

// UserHandler định nghĩa giao thức cho HTTP Delivery
type UserHandler interface {
	RegisterRoutes(e *echo.Echo, jwtSecret string)
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

// RegisterRoutes gắn các API endpoint vào Echo Router
func (h *userHandlerImpl) RegisterRoutes(e *echo.Echo, jwtSecret string) {
	// ==========================================
	// 1. API PUBLIC (Không cần Token)
	// ==========================================
	publicGroup := e.Group("/api/v1/users")
	publicGroup.POST("/register", h.register)
	publicGroup.POST("/login", h.login)
	publicGroup.POST("/refresh-token", h.refreshToken) // Cấp lại token mới bằng Refresh Token

	// Cấu hình Middleware phân giải JWT mặc định
	jwtConfig := echojwt.Config{
		SigningKey: []byte(jwtSecret),
		ErrorHandler: func(c *echo.Context, err error) error {
			return response.Error(c, http.StatusUnauthorized, "Token không hợp lệ hoặc đã hết hạn")
		},
	}

	// ==========================================
	// 2. API PRIVATE (Yêu cầu đăng nhập)
	// ==========================================
	// Bảo vệ bằng echojwt VÀ CheckBlacklist (Ngăn dùng token đã bị đăng xuất)
	protectedGroup := e.Group("/api/v1/users/me",
		echojwt.WithConfig(jwtConfig),
		middlewares.CheckBlacklist(h.useCase),
	)
	protectedGroup.GET("", h.getProfile)
	protectedGroup.POST("/logout", h.logout)

	// ==========================================
	// 3. API ADMIN (Yêu cầu quyền Quản trị)
	// ==========================================
	// adminGroup := e.Group("/api/v1/admin/users",
	// 	echojwt.WithConfig(jwtConfig),
	// 	middlewares.CheckBlacklist(h.useCase),
	// 	middlewares.RequireRole("admin"), // Chỉ cho phép Role = admin đi qua
	// )
	// adminGroup.GET("", h.adminGetUsers)
}

// --- CÁC HÀM XỬ LÝ (CONTROLLERS) ---

func (h *userHandlerImpl) register(c *echo.Context) error {
	var req domain.UserRegisterReq
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Dữ liệu JSON không hợp lệ")
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Sai định dạng dữ liệu: "+err.Error())
	}
	user, err := h.useCase.Register(c.Request().Context(), &req)
	if err != nil {
		return response.Error(c, http.StatusConflict, err.Error())
	}
	return response.Success(c, http.StatusCreated, "Đăng ký thành công", user)
}

func (h *userHandlerImpl) login(c *echo.Context) error {
	var req domain.UserLoginReq
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Dữ liệu JSON không hợp lệ")
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Sai định dạng dữ liệu: "+err.Error())
	}
	res, err := h.useCase.Login(c.Request().Context(), &req)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, err.Error())
	}
	return response.Success(c, http.StatusOK, "Đăng nhập thành công", res)
}

func (h *userHandlerImpl) refreshToken(c *echo.Context) error {
	var req domain.RefreshTokenReq
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Dữ liệu JSON không hợp lệ")
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Vui lòng cung cấp refresh_token")
	}

	res, err := h.useCase.RefreshToken(c.Request().Context(), &req)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, err.Error())
	}
	return response.Success(c, http.StatusOK, "Cấp lại token thành công", res)
}

func (h *userHandlerImpl) logout(c *echo.Context) error {
	// 1. Lấy JTI (Mã định danh token) và thời gian hết hạn từ Header
	_, _, jti, exp, err := middlewares.ExtractUserClaims(c)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, "Token không hợp lệ")
	}

	// 2. Client gửi kèm Refresh Token dưới dạng Body JSON để hệ thống xóa luôn khỏi DB
	var req domain.RefreshTokenReq
	_ = c.Bind(&req) // Bỏ qua bắt lỗi nếu client không gửi body

	// 3. Đưa Access Token vào danh sách đen & Xóa Refresh Token
	err = h.useCase.Logout(c.Request().Context(), jti, req.RefreshToken, exp)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Lỗi khi đăng xuất: "+err.Error())
	}

	return response.Success(c, http.StatusOK, "Đăng xuất thành công", nil)
}

func (h *userHandlerImpl) getProfile(c *echo.Context) error {
	// Sử dụng hàm tiện ích đã viết ở auth_middleware để lấy dữ liệu gọn gàng
	userID, role, _, _, err := middlewares.ExtractUserClaims(c)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, err.Error())
	}

	return response.Success(c, http.StatusOK, "Lấy thông tin profile thành công", map[string]any{
		"user_id": userID,
		"role":    role,
	})
}
