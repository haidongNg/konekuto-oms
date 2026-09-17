package http

import (
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/haidongNg/konekuto-oms/internal/domain"
	"github.com/haidongNg/konekuto-oms/pkg/response"
	echojwt "github.com/labstack/echo-jwt/v5" // Middleware JWT nguyên bản của Labstack
	"github.com/labstack/echo/v5"
)

type UserHandler struct {
	userUseCase domain.UserUseCase
}

// Đăng ký route và khởi tạo Handler
func NewUserHandler(e *echo.Echo, us domain.UserUseCase, jwtSecret string) {
	handler := &UserHandler{
		userUseCase: us,
	}

	// --- NHÓM API PUBLIC (KHÔNG CẦN TOKEN) ---
	publicGroup := e.Group("/api/v1/users")
	publicGroup.POST("/register", handler.Register)
	publicGroup.POST("/login", handler.Login)

	// --- NHÓM API PRIVATE (BẢO VỆ BẰNG JWT) ---
	jwtConfig := echojwt.Config{
		SigningKey: []byte(jwtSecret),
		// ErrorHandler ép lỗi của thư viện echo-jwt trả về đúng định dạng pkg/response
		ErrorHandler: func(c *echo.Context, err error) error {
			return response.Error(c, http.StatusUnauthorized, "Token không hợp lệ hoặc đã hết hạn")
		},
	}
	// Đính kèm Middleware JWT vào Route Group
	protectedGroup := e.Group("/api/v1/users/me", echojwt.WithConfig(jwtConfig))
	protectedGroup.GET("", handler.GetProfile)
}

func (h *UserHandler) Register(c *echo.Context) error {
	var req domain.UserRegisterReq

	// 1. Map body JSON vào struct
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Dữ liệu JSON không hợp lệ")
	}

	// 2. Validate dữ liệu qua pkg/validations (tags: required, email, min...)
	if err := c.Validate(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Sai định dạng dữ liệu: "+err.Error())
	}

	// 3. Gọi logic tầng UseCase
	user, err := h.userUseCase.Register(c.Request().Context(), &req)
	if err != nil {
		return response.Error(c, http.StatusConflict, err.Error())
	}

	// 4. Trả về thành công
	return response.Success(c, http.StatusCreated, "Đăng ký thành công", user)
}

func (h *UserHandler) Login(c *echo.Context) error {
	var req domain.UserLoginReq
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Dữ liệu JSON không hợp lệ")
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Sai định dạng dữ liệu: "+err.Error())
	}

	res, err := h.userUseCase.Login(c.Request().Context(), &req)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, err.Error())
	}

	return response.Success(c, http.StatusOK, "Đăng nhập thành công", res)
}

func (h *UserHandler) GetProfile(c *echo.Context) error {
	// Middleware echo-jwt mặc định lưu token đã parse vào context với key "user"
	userToken := c.Get("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)

	// Trích xuất thông tin
	userID := claims["user_id"].(string)
	role := claims["role"].(string)

	return response.Success(c, http.StatusOK, "Lấy thông tin thành công", map[string]string{
		"user_id": userID,
		"role":    role,
	})
}
