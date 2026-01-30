package grpcserv

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/Okenamay/securawr/gen/proto"
	"github.com/Okenamay/securawr/internal/server/auth"
	"github.com/Okenamay/securawr/internal/server/auth/token"
	"github.com/Okenamay/securawr/internal/server/config"
	"github.com/Okenamay/securawr/internal/server/handlers"
	"github.com/Okenamay/securawr/internal/server/storage"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
	conf       *config.Config
	logger     *zap.Logger
}

func New(
	conf *config.Config,
	logger *zap.Logger,
	creds credentials.TransportCredentials,
	db *storage.PostgresDB,
) (*Server, error) {

	// 1. Инициализация Token Manager (24 часа TTL)
	tokenManager := token.New(conf.JWTSecret, 24*time.Hour)

	// 2. Опции сервера (Interceptor + TLS)
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(auth.NewInterceptor(tokenManager).Unary),
	}
	if creds != nil {
		opts = append(opts, grpc.Creds(creds))
	}

	grpcSrv := grpc.NewServer(opts...)

	// 3. Создаем и регистрируем AuthHandler
	authHandler := handlers.NewAuthHandler(db, conf, logger, tokenManager)
	pb.RegisterAuthServiceServer(grpcSrv, authHandler)

	// DataService будет добавлен позже
	return &Server{
		grpcServer: grpcSrv,
		conf:       conf,
		logger:     logger,
	}, nil
}

func (s *Server) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", s.conf.GRPCPort)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.conf.GRPCPort, err)
	}

	// Graceful Shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		s.logger.Info("Shutting down server...")
		s.grpcServer.GracefulStop()
	}()

	s.logger.Info("gRPC server started", zap.String("addr", s.conf.GRPCPort))
	return s.grpcServer.Serve(s.listener)
}

func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}

// func RunServer(conf *config.Config, log *zap.Logger, creds credentials.TransportCredentials,
// 	store *storage.Storage) {

// 	lis, err := net.Listen("tcp", conf.GRPCPort)
// 	if err != nil {
// 		log.Fatal("Failed to listen", zap.String("port", conf.GRPCPort), zap.Error(err))
// 	}

// 	// Инициализируем менеджер токенов (TTL 24 часа для примера)
// 	tokenManager := token.New(conf.JWTSecret, 24*time.Hour)

// 	// Инициализируем Auth Interceptor
// 	authInterceptor := auth.NewInterceptor(tokenManager)

// 	// Добавляем интерцептор в опции сервера
// 	grpcServer := grpc.NewServer(
// 		grpc.Creds(creds),
// 		grpc.UnaryInterceptor(authInterceptor.Unary),
// 	)

// 	// Инициализируем хендлер, который реализует интерфейсы
// 	h := handlers.New(store, log, conf, tokenManager)

// 	// Регистрируем сервисы, описанные в proto/securawr.proto
// 	pb.RegisterAuthServiceServer(grpcServer, h)
// 	pb.RegisterDataServiceServer(grpcServer, h)

// 	// Graceful Shutdown
// 	go func() {
// 		sigCh := make(chan os.Signal, 1)
// 		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
// 		<-sigCh
// 		log.Info("Shutting down server...")
// 		grpcServer.GracefulStop()
// 	}()

// 	log.Info("Server listening", zap.String("address", conf.GRPCPort))
// 	if err := grpcServer.Serve(lis); err != nil {
// 		log.Fatal("Server error", zap.Error(err))
// 	}
// }
