package handlers

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
