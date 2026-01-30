package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type PostgresDB struct {
	db *sqlx.DB
}

func NewPostgresDB(db *sqlx.DB) *PostgresDB {
	return &PostgresDB{db: db}
}

// CreateUser создает нового пользователя с солями
func (r *PostgresDB) CreateUser(ctx context.Context, login, passwordHash string, authSalt, encSalt []byte) (*User, error) {
	query := `
		INSERT INTO users (login, password_hash, auth_salt, encryption_salt)
		VALUES ($1, $2, $3, $4)
		RETURNING id, login, password_hash, auth_salt, encryption_salt, created_at, updated_at
	`
	var user User
	err := r.db.GetContext(ctx, &user, query, login, passwordHash, authSalt, encSalt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

// GetUserByLogin ищет пользователя по логину
func (r *PostgresDB) GetUserByLogin(ctx context.Context, login string) (*User, error) {
	query := `SELECT * FROM users WHERE login = $1 LIMIT 1`
	var user User
	err := r.db.GetContext(ctx, &user, query, login)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

var ErrUserNotFound = errors.New("user not found")

func (r *PostgresDB) Close() error {
	return r.db.Close()
}
