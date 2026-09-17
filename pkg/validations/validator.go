package validations

import "github.com/go-playground/validator/v10"

// CustomValidator bọc thư viện go-playground/validator để tuân thủ interface của Echo
type CustomValidator struct {
	validator *validator.Validate
}

// Khởi tạo instance của validator
func NewValidator() *CustomValidator {
	return &CustomValidator{
		validator: validator.New(),
	}
}

// Validate thực thi quá trình kiểm tra các struct tag `validate`
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}
