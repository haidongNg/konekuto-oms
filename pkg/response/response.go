package response

import "github.com/labstack/echo/v5"

// Response định nghĩa cấu trúc JSON trả về chuẩn cho toàn bộ API
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"` // omitempty: tự động ẩn nếu không có dữ liệu
}

// Success trả về HTTP status 2xx kèm dữ liệu
func Success(c *echo.Context, statusCode int, message string, data interface{}) error {
	return c.JSON(statusCode, Response{
		Code:    statusCode,
		Message: message,
		Data:    data,
	})
}

// Error trả về HTTP status 4xx, 5xx và ép format lỗi
func Error(c *echo.Context, statusCode int, message string) error {
	return c.JSON(statusCode, Response{
		Code:    statusCode,
		Message: message,
		Data:    nil,
	})
}
