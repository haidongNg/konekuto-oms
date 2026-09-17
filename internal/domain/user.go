package domain

import (
	"context"
	"time"
)

// User là thực thể gốc (Entity) ánh xạ 1-1 với bảng users trong Database
type User struct {
	ID           string     `json:"id" db:"id"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"` // json:"-" để luôn giấu password khi trả về client
	FullName     string     `json:"full_name" db:"full_name"`
	PhoneNumber  string     `json:"phone_number" db:"phone_number"`
	AvatarURL    string     `json:"avatar_url" db:"avatar_url"`
	Role         string     `json:"role" db:"role"`
	Status       string     `json:"status" db:"status"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time `json:"-" db:"deleted_at"` // Hỗ trợ xóa mềm (Soft delete)
}

// UserRegisterReq là Data Transfer Object (DTO) nhận data từ Client khi Đăng ký
type UserRegisterReq struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=6"`
	FullName    string `json:"full_name" validate:"required"`
	PhoneNumber string `json:"phone_number" validate:"omitempty,numeric"` // omitempty: cho phép rỗng, nếu có phải là số
}

// UserLoginReq nhận data từ Client khi Đăng nhập
type UserLoginReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// UserLoginRes kết quả trả về khi Đăng nhập thành công
type UserLoginRes struct {
	AccessToken string `json:"access_token"`
	User        *User  `json:"user"`
}

// UserRepository hợp đồng giao tiếp với Database (Dependency Inversion)
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
}

// UserUseCase hợp đồng logic nghiệp vụ (Core Logic)
type UserUseCase interface {
	Register(ctx context.Context, req *UserRegisterReq) (*User, error)
	Login(ctx context.Context, req *UserLoginReq) (*UserLoginRes, error)
}
