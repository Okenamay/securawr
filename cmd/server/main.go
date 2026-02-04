package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
	creds, err := tlsserv.TLSInitialize(log)
	if err != nil {
		log.Fatal("Failed to initialize TLS server", zap.Error(err))
	}

	// 5. Инициализация gRPC сервера
	server, err := grpcserv.New(conf, log, creds, store)
	if err != nil {
		log.Fatal("Failed to create gRPC server", zap.Error(err))
	}

	// 6. Запуск gRPC сервера
	go func() {
		if err := server.Start(); err != nil {
			log.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// 7. Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info("Shutting down server...")
	server.Stop()
	log.Info("Server stopped")

}
