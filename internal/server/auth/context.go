package auth

import (
	"context"
	"errors"
)

type contextKey struct{}

var userIDKey = contextKey{}

var ErrNoUserInContext = errors.New("no user id in context")

// ContextWithUserID добавляет ID пользователя в контекст
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext извлекает ID пользователя из контекста
func UserIDFromContext(ctx context.Context) (string, error) {
	val := ctx.Value(userIDKey)
	if val == nil {
		return "", ErrNoUserInContext
	}
	userID, ok := val.(string)
	if !ok {
		return "", ErrNoUserInContext
	}
	return userID, nil
}
