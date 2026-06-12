package repository

import (
	"context"
	"database/sql"
	"errors"

	"hackathon-backend/internal/models"
)

// UserRepository は users テーブルへのDB操作を担当します。
// ハンドラにSQLを直接書かず、データアクセスをこの層に分けることで見通しを良くしています。
type UserRepository struct {
	DB *sql.DB
}

// Create は新しいユーザーをDBに保存します。
func (r *UserRepository) Create(ctx context.Context, name, email, passwordHash string) (models.User, error) {
	result, err := r.DB.ExecContext(
		ctx,
		`INSERT INTO users (name, email, password_hash) VALUES (?, ?, ?)`,
		name,
		email,
		passwordHash,
	)
	if err != nil {
		return models.User{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.User{}, err
	}

	return r.FindByID(ctx, id)
}

// FindByEmail はメールアドレスからユーザーを取得します。
// ログイン時にパスワードハッシュを取り出すために使います。
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (models.User, error) {
	var user models.User
	err := r.DB.QueryRowContext(
		ctx,
		`SELECT id, name, email, password_hash, created_at FROM users WHERE email = ?`,
		email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, err
	}
	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

// FindByID はIDからユーザーを取得します。
func (r *UserRepository) FindByID(ctx context.Context, id int64) (models.User, error) {
	var user models.User
	err := r.DB.QueryRowContext(
		ctx,
		`SELECT id, name, email, password_hash, created_at FROM users WHERE id = ?`,
		id,
	).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt)

	if err != nil {
		return models.User{}, err
	}
	return user, nil
}
