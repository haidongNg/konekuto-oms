package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/haidongNg/konekuto-oms/internal/domain"
	"github.com/jmoiron/sqlx"
)

type sqliteUserRepository struct {
	db *sqlx.DB
}

// Khởi tạo Repository bằng sqlx.DB
func NewSQLiteUserRepository(db *sqlx.DB) domain.UserRepository {
	return &sqliteUserRepository{db: db}
}

// Create chèn user mới vào DB. Sử dụng NamedExecContext để sqlx tự map trường của struct vào các biến :name
func (r *sqliteUserRepository) Create(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, full_name, phone_number, avatar_url, role, status, created_at, updated_at) 
		VALUES (:id, :email, :password_hash, :full_name, :phone_number, :avatar_url, :role, :status, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, u)
	return err
}

// GetByEmail tìm user. Sử dụng GetContext để sqlx tự map kết quả query vào biến user
func (r *sqliteUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	query := `SELECT id, email, password_hash, full_name, role, status FROM users WHERE email = ? AND deleted_at IS NULL`

	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		// sqlx/sql trả về sql.ErrNoRows nếu không tìm thấy data
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
