package handlers

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Okenamay/securawr/gen/proto"
	"github.com/Okenamay/securawr/internal/server/auth"
	"github.com/Okenamay/securawr/internal/server/config"
	"github.com/Okenamay/securawr/internal/server/storage"
	"github.com/google/uuid"
)

// DataHandler реализует интерфейсы gRPC сервиса работы с данными
type DataHandler struct {
	pb.UnimplementedDataServiceServer
	storage *storage.Storage
	conf    *config.Config
	logger  *zap.Logger
}

// NewDataHandler создаёт экземпляр DataHandler
func NewDataHandler(
	storage *storage.Storage,
	conf *config.Config,
	logger *zap.Logger) *DataHandler {
	return &DataHandler{
		storage: storage,
		conf:    conf,
		logger:  logger,
	}
}

// AddData добавляет новые зашифрованные данные пользователя
func (h *DataHandler) AddData(ctx context.Context, req *pb.AddDataRequest) (*pb.AddDataResponse, error) {
	// 1. Получаем UserID из контекста (извлеченного интерсептором)
	userIDStr, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}
	userID, _ := uuid.Parse(userIDStr)

	// 2. Проверяем валидность входных данных
	if len(req.EncryptedData) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty data payload")
	}

	// 3. Проверка квот
	// Сначала узнаем текущее потребление места пользователем
	currentUsage, err := h.storage.GetUserUsage(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get user usage", zap.Error(err))
		return nil, status.Error(codes.Internal, "storage error")
	}

	// Считаем размер новых данных
	newDataSize := int64(len(req.EncryptedData) + len(req.EncryptedKey))

	// Проверяем превышение лимита
	if currentUsage+newDataSize > h.conf.UserQuota {
		h.logger.Warn("user quota exceeded",
			zap.String("user_id", userID.String()),
			zap.Int64("current", currentUsage),
			zap.Int64("new", newDataSize),
			zap.Int64("limit", h.conf.UserQuota),
		)
		return nil, status.Errorf(codes.ResourceExhausted,
			"quota exceeded: current usage %d + new %d > limit %d",
			currentUsage, newDataSize, h.conf.UserQuota)
	}

	// 4. Подготовка метаданных
	metaInfo := req.MetaInfo
	if metaInfo == "" {
		// Fallback если клиент прислал пустоту (для надежности можно сделать базовый JSON)
		metaInfo = "{}"
	}

	// 5. Сохранение блоба данных
	newID := uuid.New()
	record := storage.DataRecord{
		ID:       newID,
		UserID:   userID,
		DataType: int(req.Type),
		DataBlob: req.EncryptedData,
		MetaInfo: metaInfo,
		Version:  1,
	}

	if err := h.storage.CreateDataRecord(ctx, record); err != nil {
		h.logger.Error("Failed to save data", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to save data")
	}

	h.logger.Info("Data saved successfully", zap.String("id", newID.String()), zap.String("user_id", userIDStr))

	return &pb.AddDataResponse{
		Id: newID.String(),
	}, nil
}

// ListData возвращает список файлов пользователя
func (h *DataHandler) ListData(ctx context.Context, req *pb.ListDataRequest) (*pb.ListDataResponse, error) {
	userIDStr, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	userID, _ := uuid.Parse(userIDStr)

	records, err := h.storage.ListDataRecords(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to list data", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to fetch data list")
	}

	var responseItems []*pb.DataRecordInfo
	for _, r := range records {
		// Фильтр по типу (если задан)
		if req.TypeFilter != pb.DataType_UNKNOWN && int(req.TypeFilter) != r.DataType {
			continue
		}

		responseItems = append(responseItems, &pb.DataRecordInfo{
			Id:        r.ID.String(),
			Type:      pb.DataType(r.DataType),
			MetaInfo:  r.MetaInfo,
			CreatedAt: r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.ListDataResponse{
		Items: responseItems,
	}, nil
}

// GetData возвращает содержимое файла
func (h *DataHandler) GetData(ctx context.Context, req *pb.GetDataRequest) (*pb.GetDataResponse, error) {
	userIDStr, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	userID, _ := uuid.Parse(userIDStr)
	dataID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid data ID")
	}

	record, err := h.storage.GetDataRecord(ctx, dataID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrDataNotFound) {
			return nil, status.Error(codes.NotFound, "data not found")
		}
		h.logger.Error("Failed to fetch data", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to fetch data")
	}

	return &pb.GetDataResponse{
		Id:            record.ID.String(),
		Type:          pb.DataType(record.DataType),
		EncryptedData: record.DataBlob,
		MetaInfo:      record.MetaInfo,
		CreatedAt:     timestamppb.New(record.CreatedAt),
		UpdatedAt:     timestamppb.New(record.UpdatedAt),
	}, nil
}

// DeleteData удаляет файл
func (h *DataHandler) DeleteData(ctx context.Context, req *pb.DeleteDataRequest) (*pb.DeleteDataResponse, error) {
	userIDStr, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	userID, _ := uuid.Parse(userIDStr)
	dataID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid data ID")
	}

	if err := h.storage.DeleteDataRecord(ctx, dataID, userID); err != nil {
		if errors.Is(err, storage.ErrDataNotFound) {
			return nil, status.Error(codes.NotFound, "data not found")
		}
		h.logger.Error("Failed to delete data", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to delete data")
	}

	h.logger.Info("Data deleted", zap.String("id", req.Id), zap.String("user_id", userIDStr))

	return &pb.DeleteDataResponse{
		Success: true,
	}, nil
}
