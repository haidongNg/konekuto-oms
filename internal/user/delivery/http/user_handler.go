package http

import (
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/haidongNg/konekuto-oms/internal/domain"
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
	useCase domain.UserUseCase // Phụ thuộc vào Interface UseCase
}

// NewUserHandler trả về Interface UserHandler
func NewUserHandler(us domain.UserUseCase) UserHandler {
	return &userHandlerImpl{
		useCase: us,
	}
}

// RegisterRoutes được gọi bởi Server để đăng ký các API endpoints
func (h *userHandlerImpl) RegisterRoutes(e *echo.Echo, jwtSecret string) {
	publicGroup := e.Group("/api/v1/users")
	publicGroup.POST("/register", h.register)
	publicGroup.POST("/login", h.login)

	jwtConfig := echojwt.Config{
		SigningKey: []byte(jwtSecret),
		ErrorHandler: func(c *echo.Context, err error) error {
			return response.Error(c, http.StatusUnauthorized, "Token không hợp lệ hoặc đã hết hạn")
		},
	}

	protectedGroup := e.Group("/api/v1/users/me", echojwt.WithConfig(jwtConfig))
	protectedGroup.GET("", h.getProfile)
}

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

func (h *userHandlerImpl) getProfile(c *echo.Context) error {
	userToken := c.Get("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	return response.Success(c, http.StatusOK, "Thành công", map[string]any{
		"user_id": claims["user_id"],
		"role":    claims["role"],
	})
}
