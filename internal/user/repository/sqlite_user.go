package repository

import (
	"context"
	"database/sql"
	"errors"

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
