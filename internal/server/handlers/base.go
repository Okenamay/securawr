package handlers

import (
	"context"

	"go.uber.org/zap"

	pb "github.com/Okenamay/securawr/gen/proto"
	"github.com/Okenamay/securawr/internal/server/storage"
)

// Handler реализует интерфейс gRPC сервиса
type Handler struct {
	pb.UnimplementedSecuRawrServiceServer
	storage *storage.Storage
	log     *zap.Logger
}

// New создает новый экземпляр хендлера
func New(storage *storage.Storage, log *zap.Logger) *Handler {
	return &Handler{
		storage: storage,
		log:     log,
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
