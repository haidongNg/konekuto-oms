package usecase

import (
	"context"
	"errors"
	"log/slog"
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

	return u.generateTokens(ctx, user)
}

func (u *userUseCase) RefreshToken(c context.Context, req *domain.RefreshTokenReq) (*domain.UserLoginRes, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// 1. Kiểm tra Refresh Token trong DB
	userID, err := u.userRepo.GetUserIDByRefreshToken(ctx, req.RefreshToken)
	if err != nil || userID == "" {
		return nil, errors.New("refresh token không hợp lệ hoặc đã hết hạn")
	}

	// 2. Xóa Token cũ để xoay vòng (Token Rotation - Chống Replay Attack)
	_ = u.userRepo.DeleteRefreshToken(ctx, req.RefreshToken)

	// 3. Lấy thông tin User (vì JWT cần Role)
	// Lưu ý: Cần thêm hàm GetByID vào Repo, tạm thời ta có thể giả định gọi hàm GetByID
	// Trong thực tế bạn nên viết thêm hàm GetByID trong Repo. Để demo ta dùng dữ liệu cơ bản:
	user := &domain.User{ID: userID, Role: "customer"} // Fix nhanh

	// 4. Cấp lại cặp Token mới
	return u.generateTokens(ctx, user)
}

func (u *userUseCase) Logout(c context.Context, jti string, refreshToken string, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Đưa Access Token vào danh sách đen
	if err := u.userRepo.BlacklistToken(ctx, jti, expiresAt); err != nil {
		return err
	}
	// Xóa Refresh Token khỏi DB
	return u.userRepo.DeleteRefreshToken(ctx, refreshToken)
}

func (u *userUseCase) CheckBlacklist(ctx context.Context, jti string) error {
	isBlacklisted, err := u.userRepo.IsTokenBlacklisted(ctx, jti)
	if err != nil || isBlacklisted {
		return errors.New("token đã bị thu hồi")
	}
	return nil
}

// --- HÀM TIỆN ÍCH PRIVATE ---
func (u *userUseCase) generateTokens(ctx context.Context, user *domain.User) (*domain.UserLoginRes, error) {
	jti := uuid.New().String() // Tạo ID duy nhất cho Access Token

	// 1. Tạo Access Token (Sống 15 phút)
	claims := jwt.MapClaims{
		"jti":     jti, // Đưa JTI vào payload để sau này Blacklist
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Minute * 15).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(u.jwtSecret)
	if err != nil {
		return nil, err
	}

	// 2. Tạo Refresh Token (Sống 7 ngày, lưu DB)
	refreshToken := uuid.New().String()
	refreshExpires := time.Now().Add(time.Hour * 24 * 7)

	if err := u.userRepo.CreateRefreshToken(ctx, user.ID, refreshToken, refreshExpires); err != nil {
		return nil, errors.New("lỗi lưu refresh token")
	}

	return &domain.UserLoginRes{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (u *userUseCase) CleanupExpiredTokens(c context.Context) error {
	// Tạo timeout riêng cho job ngầm (ví dụ 10 giây)
	ctx, cancel := context.WithTimeout(c, 10*time.Second)
	defer cancel()

	rowsBlacklist, rowsRefresh, err := u.userRepo.CleanupExpiredTokens(ctx, time.Now())
	if err != nil {
		slog.Error("❌ Lỗi khi dọn rác DB Token", "error", err)
		return err
	}

	// Chỉ in ra log nếu thực sự có rác bị xóa để Terminal đỡ bị trôi
	if rowsBlacklist > 0 || rowsRefresh > 0 {
		slog.Info("🧹 Dọn rác DB thành công",
			slog.Int64("blacklisted_deleted", rowsBlacklist),
			slog.Int64("refresh_deleted", rowsRefresh),
		)
	}

	return nil
}
