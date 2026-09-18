package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/haidongNg/konekuto-oms/internal/domain"
	"github.com/jmoiron/sqlx"
)

// sqliteUserRepository là bản thực thi (Private)
type sqliteUserRepository struct {
	db *sqlx.DB
}

// NewSQLiteUserRepository trả về Interface domain.UserRepository
// Giúp tầng UseCase không quan tâm đến việc ta đang dùng SQLite hay Postgres
func NewSQLiteUserRepository(db *sqlx.DB) domain.UserRepository {
	return &sqliteUserRepository{db: db}
}

func (r *sqliteUserRepository) Create(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, full_name, phone_number, avatar_url, role, status, created_at, updated_at) 
		VALUES (:id, :email, :password_hash, :full_name, :phone_number, :avatar_url, :role, :status, :created_at, :updated_at)`
	_, err := r.db.NamedExecContext(ctx, query, u)
	return err
}

func (r *sqliteUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	query := `SELECT id, email, password_hash, full_name, role, status FROM users WHERE email = ? AND deleted_at IS NULL`
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Trả về nil thay vì báo lỗi nếu không tìm thấy
		}
		return nil, err
	}
	return &user, nil
}

func (r *sqliteUserRepository) CreateRefreshToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	query := `INSERT INTO refresh_tokens (token, user_id, expires_at) VALUES (?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, token, userID, expiresAt)
	return err
}

func (r *sqliteUserRepository) GetUserIDByRefreshToken(ctx context.Context, token string) (string, error) {
	var userID string
	query := `SELECT user_id FROM refresh_tokens WHERE token = ? AND expires_at > CURRENT_TIMESTAMP`
	err := r.db.GetContext(ctx, &userID, query, token)
	return userID, err
}

func (r *sqliteUserRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	query := `DELETE FROM refresh_tokens WHERE token = ?`
	_, err := r.db.ExecContext(ctx, query, token)
	return err
}

func (r *sqliteUserRepository) BlacklistToken(ctx context.Context, jti string, expiresAt time.Time) error {
	query := `INSERT INTO blacklisted_tokens (jti, expires_at) VALUES (?, ?)`
	_, err := r.db.ExecContext(ctx, query, jti, expiresAt)
	return err
}

func (r *sqliteUserRepository) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	var count int
	query := `SELECT COUNT(1) FROM blacklisted_tokens WHERE jti = ?`
	err := r.db.GetContext(ctx, &count, query, jti)
	return count > 0, err
}

func (r *sqliteUserRepository) CleanupExpiredTokens(ctx context.Context, now time.Time) (int64, int64, error) {
	// 1. Xóa Access Token hết hạn
	res1, err := r.db.ExecContext(ctx, `DELETE FROM blacklisted_tokens WHERE expires_at < ?`, now)
	if err != nil {
		return 0, 0, err
	}
	rows1, _ := res1.RowsAffected()

	// 2. Xóa Refresh Token hết hạn
	res2, err := r.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE expires_at < ?`, now)
	if err != nil {
		return 0, 0, err
	}
	rows2, _ := res2.RowsAffected()

	return rows1, rows2, nil
}
