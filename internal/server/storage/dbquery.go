package storage

import (
	"context"
	"errors"

	"go.uber.org/zap"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

// CreateUser сохраняет нового пользователя в БД. На Этапе 2 структура User уже
// должна быть заполнена хешем и солью
func (s *Storage) CreateUser(ctx context.Context, u User) error {
	query := `
		INSERT INTO users (id, login, password_hash, salt, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`
	_, err := s.Pool.Exec(ctx, query, u.ID, u.Login, u.PasswordHash, u.Salt)
	if err != nil {
		s.log.Error("Failed to create user", zap.Error(err), zap.String("login", u.Login))
		return err
	}

	s.log.Info("User created successfully", zap.String("login", u.Login), zap.String("id", u.ID.String()))
	return nil
}

// GetUserByLogin ищет пользователя по его логину
func (s *Storage) GetUserByLogin(ctx context.Context, login string) (*User, error) {
	query := `
		SELECT id, login, password_hash, salt, created_at, updated_at
		FROM users
		WHERE login = $1
	`
	var u User
	err := s.Pool.QueryRow(ctx, query, login).Scan(
		&u.ID, &u.Login, &u.PasswordHash, &u.Salt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return &u, nil
}
