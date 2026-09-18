package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/haidongNg/konekuto-oms/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

// userUseCase là struct chứa logic (Private)
type userUseCase struct {
	userRepo       domain.UserRepository // Bắt buộc dùng Interface
	contextTimeout time.Duration
	jwtSecret      []byte
}

// NewUserUseCase trả về Interface domain.UserUseCase
func NewUserUseCase(ur domain.UserRepository, timeout time.Duration, secret string) domain.UserUseCase {
	return &userUseCase{
		userRepo:       ur,
		contextTimeout: timeout,
		jwtSecret:      []byte(secret),
	}
}

func (u *userUseCase) Register(c context.Context, req *domain.UserRegisterReq) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingUser, _ := u.userRepo.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, errors.New("email đã được đăng ký")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("lỗi mã hóa mật khẩu")
	}

	now := time.Now()
	newUser := &domain.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FullName:     req.FullName,
		PhoneNumber:  req.PhoneNumber,
		AvatarURL:    "",
		Role:         "customer",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err = u.userRepo.Create(ctx, newUser); err != nil {
		return nil, err
	}
	return newUser, nil
}

func (u *userUseCase) Login(c context.Context, req *domain.UserLoginReq) (*domain.UserLoginRes, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	user, err := u.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return nil, errors.New("email hoặc mật khẩu không chính xác")
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("email hoặc mật khẩu không chính xác")
	}

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(u.jwtSecret)
	if err != nil {
		return nil, errors.New("không thể tạo access token")
	}

	return &domain.UserLoginRes{
		AccessToken: accessToken,
		User:        user,
	}, nil
}
