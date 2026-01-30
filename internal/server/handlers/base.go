package handlers

import (
	"context"
	"encoding/json"
	"errors"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Okenamay/securawr/gen/proto"
	"github.com/Okenamay/securawr/internal/server/auth"
	"github.com/Okenamay/securawr/internal/server/auth/token"
	"github.com/Okenamay/securawr/internal/server/config"
	"github.com/Okenamay/securawr/internal/server/storage"
	"github.com/google/uuid"
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

// Register регистрирует нового пользователя
func (h *Handler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.Login == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "login and password cannot be empty")
	}

	h.log.Info("Register request received", zap.String("login", req.Login))

	// 1. Проверяем, не занят ли логин
	_, err := h.storage.GetUserByLogin(ctx, req.Login)
	if err == nil {
		// Если ошибки нет, значит пользователь найден -> конфликт
		return nil, status.Error(codes.AlreadyExists, "user with this login already exists")
	}
	if !errors.Is(err, storage.ErrUserNotFound) {
		// Если ошибка не "Not Found", значит что-то сломалось в БД
		h.log.Error("Failed to check user existence", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal storage error")
	}

	// 2. Генерируем соль для аутентификации (Auth_Salt)
	authSalt, err := auth.GenerateSalt()
	if err != nil {
		h.log.Error("Failed to generate salt", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate security parameters")
	}

	// 3. Хешируем пароль (Argon2id + Salt + Server_Pepper)
	// На Этапе 2) принимаем пароль как есть, на Этапе 3 req.Password будет
	// содержать Auth_Key
	hash, err := auth.HashPassword(req.Password, authSalt, h.cfg.ServerPepper)
	if err != nil {
		h.log.Error("Failed to hash password", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to process credentials")
	}

	// 4. Создаем пользователя
	newID := uuid.New()
	newUser := storage.User{
		ID:           newID,
		Login:        req.Login,
		PasswordHash: hash,
		Salt:         authSalt,
		// EncryptionSalt пока не сохраняем (будет добавлено при реализации
		// шифрования файлов)
	}

	if err := h.storage.CreateUser(ctx, newUser); err != nil {
		h.log.Error("Failed to create user", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	h.log.Info("User registered successfully", zap.String("user_id", newID.String()))

	return &pb.RegisterResponse{
		UserId: newID.String(),
	}, nil
}

// Login аутентифицирует пользователя и выдает токен
func (h *Handler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Login == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "login and password required")
	}

	h.log.Info("Login request received", zap.String("login", req.Login))

	// 1. Ищем пользователя
	user, err := h.storage.GetUserByLogin(ctx, req.Login)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, status.Error(codes.Unauthenticated, "invalid login or password")
		}
		h.log.Error("Failed to get user", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal storage error")
	}

	// 2. Проверяем пароль
	// Используем перец из конфига и хеш из БД (в котором уже зашита соль и
	// параметры Argon2)
	match, err := auth.VerifyPassword(req.Password, h.cfg.ServerPepper, user.PasswordHash)
	if err != nil {
		h.log.Error("Password verification error", zap.Error(err))
		return nil, status.Error(codes.Internal, "authentication check failed")
	}

	if !match {
		// Намеренно возвращаем такую же ошибку, как если бы пользователь не
		// был найден
		return nil, status.Error(codes.Unauthenticated, "invalid login or password")
	}

	// 3. Генерируем JWT
	tokenString, err := h.tokenManager.Generate(user.ID.String())
	if err != nil {
		h.log.Error("Failed to generate token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate session token")
	}

	h.log.Info("User logged in successfully", zap.String("user_id", user.ID.String()))

	return &pb.LoginResponse{
		Token: tokenString,
	}, nil
}

// Реализация DataService

// SaveData сохраняет данные пользователя
func (h *Handler) SaveData(ctx context.Context, req *pb.SaveDataRequest) (*pb.SaveDataResponse, error) {
	// 1. Авторизация: Получаем UserID из контекста
	userIDStr, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	userID, _ := uuid.Parse(userIDStr)

	// 2. Валидация
	if len(req.Data) == 0 {
		return nil, status.Error(codes.InvalidArgument, "data cannot be empty")
	}

	// 3. Подготовка метаданных
	meta := storage.DataMeta{
		Name:        req.Name,
		Description: req.Description,
		Filename:    req.Name, // Для простоты дублируем имя в filename
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		h.log.Error("Failed to marshal metadata", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to process metadata")
	}

	// 4. Сохранение
	newID := uuid.New()
	record := storage.DataRecord{
		ID:       newID,
		UserID:   userID,
		DataType: int(req.Type),
		DataBlob: req.Data,
		MetaInfo: string(metaJSON),
	}

	if err := h.storage.CreateDataRecord(ctx, record); err != nil {
		return nil, status.Error(codes.Internal, "failed to save data")
	}

	h.log.Info("Data saved successfully", zap.String("id", newID.String()), zap.String("user_id", userIDStr))

	return &pb.SaveDataResponse{
		Id: newID.String(),
	}, nil
}

// ListData возвращает список файлов пользователя
func (h *Handler) ListData(ctx context.Context, req *pb.ListDataRequest) (*pb.ListDataResponse, error) {
	userIDStr, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	userID, _ := uuid.Parse(userIDStr)

	records, err := h.storage.ListDataRecords(ctx, userID)
	if err != nil {
		h.log.Error("Failed to list data", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to fetch data list")
	}

	var responseItems []*pb.DataMetadata
	for _, r := range records {
		// Фильтр по типу (если задан)
		if req.TypeFilter != pb.DataType_DATA_TYPE_UNSPECIFIED && int(req.TypeFilter) != r.DataType {
			continue
		}

		// Десериализация метаданных
		var meta storage.DataMeta
		_ = json.Unmarshal([]byte(r.MetaInfo), &meta) // Игнорируем ошибку, чтобы не ломать весь список из-за одного битого JSON

		responseItems = append(responseItems, &pb.DataMetadata{
			Id:          r.ID.String(),
			Type:        pb.DataType(r.DataType),
			Name:        meta.Name,
			Description: meta.Description,
			CreatedAt:   timestamppb.New(r.CreatedAt),
			UpdatedAt:   timestamppb.New(r.UpdatedAt),
		})
	}

	return &pb.ListDataResponse{
		Items: responseItems,
	}, nil
}

// GetData возвращает содержимое файла
func (h *Handler) GetData(ctx context.Context, req *pb.GetDataRequest) (*pb.GetDataResponse, error) {
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
		return nil, status.Error(codes.Internal, "failed to fetch data")
	}

	var meta storage.DataMeta
	_ = json.Unmarshal([]byte(record.MetaInfo), &meta)

	return &pb.GetDataResponse{
		Id:          record.ID.String(),
		Type:        pb.DataType(record.DataType),
		Data:        record.DataBlob,
		Name:        meta.Name,
		Description: meta.Description,
		CreatedAt:   timestamppb.New(record.CreatedAt),
		UpdatedAt:   timestamppb.New(record.UpdatedAt),
	}, nil
}

// DeleteData удаляет файл
func (h *Handler) DeleteData(ctx context.Context, req *pb.DeleteDataRequest) (*pb.DeleteDataResponse, error) {
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
		return nil, status.Error(codes.Internal, "failed to delete data")
	}

	h.log.Info("Data deleted", zap.String("id", req.Id), zap.String("user_id", userIDStr))

	return &pb.DeleteDataResponse{
		Success: true,
	}, nil
}
