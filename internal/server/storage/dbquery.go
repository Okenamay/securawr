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

// Здесь будут методы работы с БД
// Используем ресивер (s *Storage)

// CreateUser - пример метода (заготовка для Этапа 2)
func (s *Storage) CreateUser(ctx context.Context, u User) (int64, error) {
	// TODO: Реализовать INSERT запрос
	s.log.Debug("CreateUser called", zap.String("login", u.Login))
	return 0, nil
}

// GetUserByLogin - пример метода (заготовка для Этапа 2)
func (s *Storage) GetUserByLogin(ctx context.Context, login string) (*User, error) {
	// TODO: Реализовать SELECT запрос
	return nil, ErrUserNotFound
}
