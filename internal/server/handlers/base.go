package handlers

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Okenamay/securawr/gen/proto"
	"github.com/Okenamay/securawr/internal/server/auth/token"
	"github.com/Okenamay/securawr/internal/server/config"
	"github.com/Okenamay/securawr/internal/server/storage"
)

// Handler реализует интерфейсы gRPC сервисов (AuthService и DataService)
type Handler struct {
	pb.UnimplementedAuthServiceServer
	pb.UnimplementedDataServiceServer

	storage      *storage.Storage
	log          *zap.Logger
	cfg          *config.Config
	tokenManager *token.Manager
}

// New создает новый экземпляр хендлера
func New(storage *storage.Storage, log *zap.Logger, cfg *config.Config, tm *token.Manager) *Handler {
	return &Handler{
		storage:      storage,
		log:          log,
		cfg:          cfg,
		tokenManager: tm,
	}
}

// Ping - реализация метода Ping
// Используем ресивер (h *Handler)
func (h *Handler) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	h.log.Info("Ping received", zap.String("msg", req.Message))
	return &pb.PingResponse{
		Message: "Pong: " + req.Message,
		Version: "v0.0.1",
	}, nil
}

// Реализация AuthService

func (h *Handler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// Пока заглушка
	h.log.Info("Register request received", zap.String("login", req.Login))
	return nil, status.Errorf(codes.Unimplemented, "method Register not implemented")
}

func (h *Handler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	h.log.Info("Login request received", zap.String("login", req.Login))
	return nil, status.Errorf(codes.Unimplemented, "method Login not implemented")
}

// Реализация DataService

func (h *Handler) SaveData(ctx context.Context, req *pb.SaveDataRequest) (*pb.SaveDataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SaveData not implemented")
}

func (h *Handler) ListData(ctx context.Context, req *pb.ListDataRequest) (*pb.ListDataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListData not implemented")
}

func (h *Handler) GetData(ctx context.Context, req *pb.GetDataRequest) (*pb.GetDataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetData not implemented")
}

func (h *Handler) DeleteData(ctx context.Context, req *pb.DeleteDataRequest) (*pb.DeleteDataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteData not implemented")
}
