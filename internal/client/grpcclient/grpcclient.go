package grpcclient

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewClient устанавливает соединение с gRPC сервером по указанному адресу. На
// данном этапе используется insecure соединение (без TLS)
func NewClient(address string) (*grpc.ClientConn, error) {
	// Используем опцию insecure.NewCredentials() для отключения проверки
	// сертификатов во время отладки. На Этапе 3 здесь будет загрузка TLS
	// сертификатов
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// Создаем клиентское соединение
	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc client connection: %w", err)
	}

	return conn, nil
}
