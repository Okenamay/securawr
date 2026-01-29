package tlsserv

import (
	"crypto/tls"

	"github.com/Okenamay/securawr/internal/server/config"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
	"google.golang.org/grpc/credentials"
)

// TLSInitialize настраивает TransportCredentials с использованием autocert для автоматического получения сертификатов.
func TLSInitialize(conf *config.Config, log *zap.Logger) (creds credentials.TransportCredentials, err error) {
	// Конфигурация менеджера autocert
	manager := &autocert.Manager{
		// Директория для кэширования сертификатов
		Cache: autocert.DirCache("certs"),
		// Функция, принимающая Terms of Service
		Prompt: autocert.AcceptTOS,
		// В продакшене рекомендуется ограничивать список хостов через HostPolicy
		// HostPolicy = nil, то есть, разрешены все хосты
	}

	// Создаем TLS конфигурацию на базе менеджера
	tlsConfig := manager.TLSConfig()

	// Принудительно устанавливаем версию TLS 1.3
	tlsConfig.MinVersion = tls.VersionTLS13

	// Создаем gRPC учетные данные (creds) на основе TLS конфигурации
	creds = credentials.NewTLS(tlsConfig)

	log.Info("TLS configured with autocert (ACME)")
	return creds, nil
}
