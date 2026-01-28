package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/Okenamay/securawr/gen/proto"
	logger "github.com/Okenamay/securawr/internal/logger/zap"
	"github.com/Okenamay/securawr/internal/server/config"
	"github.com/Okenamay/securawr/internal/server/handlers"
	"github.com/Okenamay/securawr/internal/server/storage"
	"github.com/Okenamay/securawr/internal/version"
)

func main() {
	// 1. Загрузка конфигурации
	cfg, err := config.InitConfig()
	if err != nil {
		// Используем стандартный вывод, так как логгер еще не инициализирован
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// 2. Инициализация логгера
	log, err := logger.New(cfg.LogLevel)
	if err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}
	defer log.Sync()

	// Используем version.Version и version.BuildDate
	log.Info("Starting SecuRawr Server",
		zap.String("version", version.Version),
		zap.String("date", version.BuildDate),
		zap.String("config_port", cfg.GRPCPort),
	)

	// 3. Инициализация БД и миграции
	store, err := storage.New(cfg.DSN, log)
	if err != nil {
		log.Fatal("Failed to initialize storage", zap.Error(err))
	}
	defer store.Close()

	// 4. Настройка TLS 1.3
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		log.Fatal("Failed to load TLS keys", zap.Error(err))
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}
	creds := credentials.NewTLS(tlsConfig)

	// 5. Запуск gRPC сервера
	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatal("Failed to listen", zap.String("port", cfg.GRPCPort), zap.Error(err))
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
	)

	h := handlers.New(store, log)
	pb.RegisterSecuRawrServiceServer(grpcServer, h)

	// Graceful Shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Info("Shutting down server...")
		grpcServer.GracefulStop()
	}()

	log.Info("Server listening", zap.String("address", cfg.GRPCPort))
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal("Server error", zap.Error(err))
	}
}
