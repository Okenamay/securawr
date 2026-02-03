package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/Okenamay/securawr/internal/server/auth/token"
)

// Interceptor содержит зависимости для проверки авторизации
type Interceptor struct {
	tokenManager *token.Manager
}

// NewInterceptor создает новый экземпляр интерцептора
func NewInterceptor(tm *token.Manager) *Interceptor {
	return &Interceptor{
		tokenManager: tm,
	}
}

// Unary возвращает ServerInterceptor для проверки JWT токенов
func (i *Interceptor) Unary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// 1. Список публичных методов, не требующих авторизации
	publicMethods := map[string]bool{
		"/securawr.AuthService/Register": true,
		"/securawr.AuthService/Login":    true,
		"/securawr.AuthService/Ping":     true,
	}

	// Если метод публичный, пропускаем запрос дальше без проверок
	if publicMethods[info.FullMethod] {
		return handler(ctx, req)
	}

	// 2. Извлекаем метаданные (заголовки)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	values := md["authorization"]
	if len(values) == 0 {
		return nil, status.Error(codes.Unauthenticated, "authorization token is not provided")
	}

	// 3. Парсим Bearer токен
	authHeader := values[0]
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, status.Error(codes.Unauthenticated, "invalid auth header format")
	}

	accessToken := parts[1]

	// 4. Валидируем токен
	userID, err := i.tokenManager.Validate(accessToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "access token is invalid: %v", err)
	}

	// 5. Кладем userID в контекст и вызываем реальный хендлер
	newCtx := ContextWithUserID(ctx, userID)
	return handler(newCtx, req)
}
