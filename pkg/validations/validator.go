package validations

import (
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
)

// customValidator được viết chữ thường để ẩn (private) khỏi các package khác
type customValidator struct {
	validator *validator.Validate
}

// NewValidator là Factory function, trả về Interface echo.Validator
func NewValidator() echo.Validator {
	return &customValidator{
		validator: validator.New(),
	}
}

// Validate thực thi hàm bắt buộc của interface echo.Validator
func (cv *customValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}
