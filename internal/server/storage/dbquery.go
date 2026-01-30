package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// CreateUser создает нового пользователя с солями
func (s *Storage) CreateUser(ctx context.Context, login, passwordHash string, authSalt, encSalt []byte) (*User, error) {
	query := `
		INSERT INTO users (login, password_hash, auth_salt, encryption_salt)
		VALUES ($1, $2, $3, $4)
		RETURNING id, login, password_hash, auth_salt, encryption_salt, created_at, updated_at
	`
	var user User
	err := s.db.GetContext(ctx, &user, query, login, passwordHash, authSalt, encSalt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

// GetUserByLogin ищет пользователя по логину
func (s *Storage) GetUserByLogin(ctx context.Context, login string) (*User, error) {
	query := `SELECT * FROM users WHERE login = $1 LIMIT 1`
	var user User
	err := s.db.GetContext(ctx, &user, query, login)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

var ErrUserNotFound = errors.New("user not found")

func (s *Storage) Close() error {
	return s.db.Close()
}
