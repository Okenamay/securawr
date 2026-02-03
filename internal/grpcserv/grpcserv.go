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
	db *storage.Storage,
) (*Server, error) {

	// 1. Инициализация Token Manager (24 часа TTL)
	tokenManager := token.New(conf.JWTSecret, 24*time.Hour)

	// Рассчитываем максимальный размер сообщения: лимит файла + 5 МБ на метаданные/оверхед
	maxMsgSize := int(conf.MaxFileSize) + (5 * 1024 * 1024)

	// 2. Опции сервера (Interceptor + TLS + Limits)
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(auth.NewInterceptor(tokenManager).Unary),
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
	}
	if creds != nil {
		opts = append(opts, grpc.Creds(creds))
	}

	grpcSrv := grpc.NewServer(opts...)

	// 3. Создаем и регистрируем AuthHandler
	authHandler := handlers.NewAuthHandler(db, conf, logger, tokenManager)
	pb.RegisterAuthServiceServer(grpcSrv, authHandler)

	// 4. Создаем и регистрируем DataHandler (с проверкой квот)
	dataHandler := handlers.NewDataHandler(db, conf, logger)
	pb.RegisterDataServiceServer(grpcSrv, dataHandler)

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
