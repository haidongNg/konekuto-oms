package validations

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

// customValidator được viết chữ thường để ẩn (private) khỏi các package khác
type customValidator struct {
	validator *validator.Validate
}

// NewValidator là Factory function, trả về Interface fiber.Validator
func NewValidator() fiber.StructValidator {
	return &customValidator{
		validator: validator.New(),
	}
}

// Validate thực thi hàm bắt buộc của interface fiber.StructValidator
func (cv *customValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}
