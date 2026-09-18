package domain

import (
	"context"
	"time"
)

// User là Entity ánh xạ trực tiếp với Database
type User struct {
	ID           string     `json:"id" db:"id"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"`
	FullName     string     `json:"full_name" db:"full_name"`
	PhoneNumber  string     `json:"phone_number" db:"phone_number"`
	AvatarURL    string     `json:"avatar_url" db:"avatar_url"`
	Role         string     `json:"role" db:"role"`
	Status       string     `json:"status" db:"status"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time `json:"-" db:"deleted_at"`
}

type UserRegisterReq struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=6"`
	FullName    string `json:"full_name" validate:"required"`
	PhoneNumber string `json:"phone_number" validate:"omitempty,numeric"`
}

type UserLoginReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type UserLoginRes struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
}

type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// =======================================
// INTERFACES (Hợp đồng Kiến trúc)
// =======================================

// UserRepository định nghĩa các thao tác với DB
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)

	CreateRefreshToken(ctx context.Context, userID, token string, expiresAt time.Time) error
	GetUserIDByRefreshToken(ctx context.Context, token string) (string, error)
	DeleteRefreshToken(ctx context.Context, token string) error

	BlacklistToken(ctx context.Context, jti string, expiresAt time.Time) error
	IsTokenBlacklisted(ctx context.Context, jti string) (bool, error)

	CleanupExpiredTokens(ctx context.Context, now time.Time) (int64, int64, error)
}

// UserUseCase định nghĩa logic nghiệp vụ (Core Logic)
type UserUseCase interface {
	Register(ctx context.Context, req *UserRegisterReq) (*User, error)
	Login(ctx context.Context, req *UserLoginReq) (*UserLoginRes, error)
	RefreshToken(ctx context.Context, req *RefreshTokenReq) (*UserLoginRes, error)
	Logout(ctx context.Context, jti string, refreshToken string, expiresAt time.Time) error
	CheckBlacklist(ctx context.Context, jti string) error
	CleanupExpiredTokens(ctx context.Context) error
}
