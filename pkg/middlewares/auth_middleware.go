package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/haidongNg/konekuto-oms/pkg/response"
	"github.com/labstack/echo/v5"
)

// ==========================================
// 1. INTERFACE CHO MIDDLEWARE
// ==========================================

// BlacklistChecker là một Interface siêu nhỏ (Interface Segregation Principle).
// Nó giúp Middleware có thể gọi hàm CheckBlacklist của Tầng UseCase
// mà không cần phải import trực tiếp toàn bộ UserUseCase (tránh vòng lặp phụ thuộc).
type BlacklistChecker interface {
	CheckBlacklist(ctx context.Context, jti string) error
}

// ==========================================
// 2. CÁC HÀM TIỆN ÍCH TRÍCH XUẤT (EXTRACTOR)
// ==========================================

// ExtractUserClaims trích xuất các thông tin quan trọng từ JWT Token.
// Hàm này phải được gọi SAU middleware echojwt (nghĩa là token đã được xác thực hợp lệ).
func ExtractUserClaims(c *echo.Context) (userID string, role string, jti string, exp time.Time, err error) {
	// "user" là key mặc định mà thư viện echojwt dùng để lưu token vào Context
	userToken, ok := c.Get("user").(*jwt.Token)
	if !ok || userToken == nil {
		return "", "", "", time.Time{}, echo.NewHTTPError(http.StatusUnauthorized, "Không tìm thấy token")
	}

	claims, ok := userToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", "", time.Time{}, echo.NewHTTPError(http.StatusUnauthorized, "Định dạng token không hợp lệ")
	}

	// Trích xuất dữ liệu an toàn
	userID, _ = claims["user_id"].(string)
	role, _ = claims["role"].(string)
	jti, _ = claims["jti"].(string)

	// Ép kiểu thời gian hết hạn (exp) từ float64 sang time.Time
	expFloat, _ := claims["exp"].(float64)
	exp = time.Unix(int64(expFloat), 0)

	return userID, role, jti, exp, nil
}

// ==========================================
// 3. CÁC MIDDLEWARE BẢO MẬT
// ==========================================

// RequireRole (RBAC): Kiểm tra xem người dùng có Role nằm trong danh sách cho phép không.
// Ví dụ sử dụng ở Route: middlewares.RequireRole("admin", "manager")
func RequireRole(allowedRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// Lấy Role từ token hiện tại
			_, role, _, _, err := ExtractUserClaims(c)
			if err != nil {
				return response.Error(c, http.StatusUnauthorized, "Vui lòng đăng nhập")
			}

			// So sánh với danh sách Role được cấp phép
			for _, allowedRole := range allowedRoles {
				if role == allowedRole {
					return next(c) // Đi tiếp vào Handler (Controller)
				}
			}

			// Chặn lại nếu không đủ thẩm quyền
			return response.Error(c, http.StatusForbidden, "Bạn không có đặc quyền truy cập tài nguyên này")
		}
	}
}

// CheckBlacklist: Kiểm tra xem Access Token này đã bị Đăng xuất (Logout) hay chưa.
func CheckBlacklist(checker BlacklistChecker) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// Trích xuất JTI (ID duy nhất của Token)
			_, _, jti, _, err := ExtractUserClaims(c)
			if err != nil {
				return response.Error(c, http.StatusUnauthorized, "Token không hợp lệ")
			}

			// Đưa xuống DB (thông qua UseCase) để check xem JTI này có nằm trong Blacklist không
			if err := checker.CheckBlacklist(c.Request().Context(), jti); err != nil {
				// Nếu có lỗi -> Token đã bị thu hồi
				return response.Error(c, http.StatusUnauthorized, "Phiên đăng nhập đã bị đăng xuất hoặc thu hồi")
			}

			// An toàn -> Đi tiếp
			return next(c)
		}
	}
}
