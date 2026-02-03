package auth

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/Okenamay/securawr/internal/server/auth/token"
)

func TestInterceptor_Unary(t *testing.T) {
	// 1. Подготовка зависимостей
	// Создаем настоящий tokenManager, так как он не зависит от внешних ресурсов
	secret := "test_secret_key"
	ttl := time.Hour
	tm := token.New(secret, ttl)

	// Генерируем валидный токен для тестов
	validUserID := "user-uuid-123"
	validToken, err := tm.Generate(validUserID)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	interceptor := NewInterceptor(tm)

	// Фиктивный хендлер, который возвращает успех и проверяет UserID в контексте (если нужно)
	mockHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	tests := []struct {
		name         string
		method       string
		md           metadata.MD
		expectCode   codes.Code
		expectUserID string // Если пусто, не проверяем UserID в контексте
	}{
		{
			name:       "Public method (Login) - No token required",
			method:     "/securawr.AuthService/Login",
			md:         nil, // Без метаданных
			expectCode: codes.OK,
		},
		{
			name:       "Public method (Register) - No token required",
			method:     "/securawr.AuthService/Register",
			md:         metadata.Pairs("authorization", "Bearer invalid"), // Даже с плохим токеном пустит
			expectCode: codes.OK,
		},
		{
			name:       "Protected method - No metadata",
			method:     "/securawr.DataService/AddData",
			md:         nil,
			expectCode: codes.Unauthenticated,
		},
		{
			name:       "Protected method - No authorization header",
			method:     "/securawr.DataService/AddData",
			md:         metadata.Pairs("content-type", "application/grpc"),
			expectCode: codes.Unauthenticated,
		},
		{
			name:       "Protected method - Invalid header format (No Bearer)",
			method:     "/securawr.DataService/AddData",
			md:         metadata.Pairs("authorization", validToken),
			expectCode: codes.Unauthenticated,
		},
		{
			name:       "Protected method - Invalid header format (Wrong prefix)",
			method:     "/securawr.DataService/AddData",
			md:         metadata.Pairs("authorization", "Basic "+validToken),
			expectCode: codes.Unauthenticated,
		},
		{
			name:       "Protected method - Invalid token (bad signature)",
			method:     "/securawr.DataService/AddData",
			md:         metadata.Pairs("authorization", "Bearer bad.token.signature"),
			expectCode: codes.Unauthenticated,
		},
		{
			name:         "Protected method - Valid token",
			method:       "/securawr.DataService/AddData",
			md:           metadata.Pairs("authorization", "Bearer "+validToken),
			expectCode:   codes.OK,
			expectUserID: validUserID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Формируем контекст
			ctx := context.Background()
			if tt.md != nil {
				ctx = metadata.NewIncomingContext(ctx, tt.md)
			}

			// Формируем Info о методе
			info := &grpc.UnaryServerInfo{
				FullMethod: tt.method,
			}

			// Если мы ожидаем проверку UserID, подменяем хендлер
			currentHandler := mockHandler
			if tt.expectUserID != "" {
				currentHandler = func(ctx context.Context, req interface{}) (interface{}, error) {
					// Проверяем, что ID пользователя попал в контекст
					uid, err := UserIDFromContext(ctx)
					if err != nil {
						t.Errorf("Handler: failed to get user id from context: %v", err)
					}
					if uid != tt.expectUserID {
						t.Errorf("Handler: expected userID %s, got %s", tt.expectUserID, uid)
					}
					return "success", nil
				}
			}

			// Вызов тестируемого метода
			resp, err := interceptor.Unary(ctx, nil, info, currentHandler)

			// Проверка кода возврата (gRPC status)
			if status.Code(err) != tt.expectCode {
				t.Errorf("Unary() error code = %v, want %v. Error: %v", status.Code(err), tt.expectCode, err)
			}

			// Если ожидался успех, проверяем, что ответ не nil
			if tt.expectCode == codes.OK && resp != "success" {
				t.Errorf("Unary() response = %v, want 'success'", resp)
			}
		})
	}
}
