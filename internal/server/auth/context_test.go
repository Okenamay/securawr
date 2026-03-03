package auth

import (
	"context"
	"testing"
)

func TestContextUserID(t *testing.T) {
	t.Run("Successfully set and retrieve UserID", func(t *testing.T) {
		expectedID := "user-uuid-1234"
		ctx := context.Background()

		// 1. Добавляем ID
		ctx = ContextWithUserID(ctx, expectedID)

		// 2. Извлекаем ID
		actualID, err := UserIDFromContext(ctx)

		// 3. Проверяем
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if actualID != expectedID {
			t.Errorf("Expected UserID %q, got %q", expectedID, actualID)
		}
	})

	t.Run("Error when UserID is missing", func(t *testing.T) {
		ctx := context.Background()

		// Пытаемся извлечь из пустого контекста
		id, err := UserIDFromContext(ctx)

		if err != ErrNoUserInContext {
			t.Errorf("Expected error ErrNoUserInContext, got %v", err)
		}
		if id != "" {
			t.Errorf("Expected empty ID, got %q", id)
		}
	})

	t.Run("Error when UserID has wrong type", func(t *testing.T) {
		// Поскольку мы находимся внутри пакета auth, мы имеем доступ к приватному
		// ключу userIDKey. Мы можем намеренно положить туда данные неверного типа (int).
		ctx := context.WithValue(context.Background(), userIDKey, 12345)

		id, err := UserIDFromContext(ctx)

		// Ожидаем ошибку, так как type assertion val.(string) провалится
		if err != ErrNoUserInContext {
			t.Errorf("Expected ErrNoUserInContext due to type mismatch, got %v", err)
		}
		if id != "" {
			t.Errorf("Expected empty ID, got %q", id)
		}
	})
}
