package usecase

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
	"github.com/haidongNg/konekuto-oms/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type userUseCase struct {
	userRepo       domain.UserRepository
	contextTimeout time.Duration
	jwtSecret      []byte
}

func NewUserUseCase(ur domain.UserRepository, timeout time.Duration, secret string) domain.UserUseCase {
	return &userUseCase{
		userRepo:       ur,
		contextTimeout: timeout,
		jwtSecret:      []byte(secret),
	}
}

func (u *userUseCase) Register(c context.Context, req *domain.UserRegisterReq) (*domain.User, error) {
	// 1. Áp dụng timeout context để tránh request treo vĩnh viễn
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// 2. Check trùng Email
	existingUser, _ := u.userRepo.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, errors.New("email đã được đăng ký")
	}

	// 3. Băm (Hash) mật khẩu bằng bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("lỗi mã hóa mật khẩu")
	}

	// 4. Khởi tạo Entity
	now := time.Now()
	newUser := &domain.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FullName:     req.FullName,
		PhoneNumber:  req.PhoneNumber,
		AvatarURL:    "",
		Role:         "customer", // Mặc định role là customer
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// 5. Lưu vào DB
	err = u.userRepo.Create(ctx, newUser)
	if err != nil {
		return nil, err
	}
	return newUser, nil
}

func (u *userUseCase) Login(c context.Context, req *domain.UserLoginReq) (*domain.UserLoginRes, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// 1. Lấy thông tin User từ DB
	user, err := u.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return nil, errors.New("email hoặc mật khẩu không chính xác") // Che giấu lỗi thật để bảo mật
	}

	// 2. So sánh mật khẩu client gửi với hash trong DB
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("email hoặc mật khẩu không chính xác")
	}

	// 3. Tạo JWT Claims chứa thông tin định danh
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 72).Unix(), // Hết hạn sau 72h
		"iat":     time.Now().Unix(),
	}

	// 4. Ký (Sign) token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(u.jwtSecret)
	if err != nil {
		return nil, errors.New("không thể tạo access token")
	}

	// 5. Trả về cho Client
	return &domain.UserLoginRes{
		AccessToken: accessToken,
		User:        user,
	}, nil
}
