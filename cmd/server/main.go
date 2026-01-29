package main

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/Okenamay/securawr/internal/grpcserv"
	logger "github.com/Okenamay/securawr/internal/logger/zap"
	"github.com/Okenamay/securawr/internal/server/config"
	"github.com/Okenamay/securawr/internal/server/storage"
	"github.com/Okenamay/securawr/internal/tlsserv"
	"github.com/Okenamay/securawr/internal/version"
)

func main() {
	// 1. Загрузка конфигурации
	conf, err := config.InitConfig()
	if err != nil {
		// Используем стандартный вывод, так как логгер еще не инициализирован
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// 2. Инициализация логгера
	log, err := logger.New(conf.LogLevel)
	if err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}
	defer log.Sync()

	// Используем version.Version и version.BuildDate
	log.Info("Starting SecuRawr Server",
		zap.String("version", version.Version),
		zap.String("date", version.BuildDate),
		zap.String("config_port", conf.GRPCPort),
	)

	// 3. Инициализация БД и миграции
	store, err := storage.New(conf, log)
	if err != nil {
		log.Fatal("Failed to initialize storage", zap.Error(err))
	}
	defer store.Close()

	// 4. Запуск TLS 1.3

	creds, err := tlsserv.TLSInitialize(conf, log)
	if err != nil {
		log.Fatal("Failed to initialize TLS server", zap.Error(err))
	}

	// 5. Запуск gRPC сервера
	grpcserv.RunServer(conf, log, creds, store)
}
