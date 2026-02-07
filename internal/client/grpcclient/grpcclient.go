package grpcclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// TokenProvider — функция, возвращающая текущий токен авторизации
type TokenProvider func() string

// NewClient устанавливает соединение с gRPC сервером
// tokenProvider используется для внедрения JWT-токена в заголовки запросов
func NewClient(address string, tokenProvider TokenProvider) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// Если провайдер токена передан, подключаем интерцептор
	if tokenProvider != nil {
		opts = append(opts, grpc.WithUnaryInterceptor(authInterceptor(tokenProvider)))
	}

	// Создаем клиентское соединение
	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc client connection: %w", err)
	}

	return conn, nil
}

// authInterceptor создает Unary Client Interceptor, который добавляет
// заголовок Authorization
func authInterceptor(tokenProvider TokenProvider) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		token := tokenProvider()

		// Добавляем токен только если он есть
		if token != "" {
			// metadata.AppendToOutgoingContext добавляет метаданные к
			// контексту, уходящему на сервер
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		}

		// Вызываем реальный метод
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
