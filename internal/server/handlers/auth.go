package handlers

import (
	"context"
	"crypto/rand"
	"errors"

	pb "github.com/Okenamay/securawr/gen/proto"
	"github.com/Okenamay/securawr/internal/crypto"
	"github.com/Okenamay/securawr/internal/server/auth"
	"github.com/Okenamay/securawr/internal/server/auth/token"
	"github.com/Okenamay/securawr/internal/server/config"
	"github.com/Okenamay/securawr/internal/server/storage"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthHandler реализует интерфейсы gRPC сервиса аутентификации.
type AuthHandler struct {
	pb.UnimplementedAuthServiceServer
	storage *storage.Storage
	conf    *config.Config
	logger  *zap.Logger
	tokman  *token.Manager
}

// NewAuthHandler создаёт экземпляр AuthHandler
func NewAuthHandler(
	storage *storage.Storage,
	conf *config.Config,
	logger *zap.Logger,
	tokman *token.Manager) *AuthHandler {
	return &AuthHandler{
		storage: storage,
		conf:    conf,
		logger:  logger,
		tokman:  tokman,
	}
}

// GetAuthParams возвращает соль для аутентификации
func (h *AuthHandler) GetAuthParams(ctx context.Context, req *pb.AuthParamsRequest) (*pb.AuthParamsResponse, error) {
	if req.Login == "" {
		return nil, status.Error(codes.InvalidArgument, "login is required")
	}

	user, err := h.storage.GetUserByLogin(ctx, req.Login)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			// Защита от User Enumeration: возвращаем фейковую соль
			fakeSalt := make([]byte, 16)
			// Используем криптостойкий рандом для фейковой соли
			if _, rErr := rand.Read(fakeSalt); rErr != nil {
				return nil, status.Error(codes.Internal, "internal error")
			}
			return &pb.AuthParamsResponse{AuthSalt: fakeSalt}, nil
		}
		h.logger.Error("failed to get user", zap.Error(err), zap.String("login", req.Login))
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.AuthParamsResponse{AuthSalt: user.AuthSalt}, nil
}

// Register регистрирует пользователя
// Клиент присылает Auth_Key и соли, полученные или сгенерированные для этого пользователя
func (h *AuthHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.Login == "" ||
		len(req.AuthKey) != crypto.KeyLen ||
		len(req.AuthSalt) != crypto.SaltLen ||
		len(req.EncryptionSalt) != crypto.SaltLen {
		return nil, status.Error(codes.InvalidArgument, "missing required registration fields")
	}

	// 1. Проверяем, не занят ли логин
	_, err := h.storage.GetUserByLogin(ctx, req.Login)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "user already exists")
	}
	if !errors.Is(err, storage.ErrUserNotFound) {
		h.logger.Error("check user existence failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	// 2. Хешируем полученный Auth_Key с Server_Pepper
	// Получаем Auth_Hash
	finalHash, err := auth.HashAuthKey(req.AuthKey, h.conf.ServerPepper)
	if err != nil {
		h.logger.Error("hashing Auth_Key failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "crypto error")
	}

	// 3. Сохраняем пользователя в БД
	// PasswordHash в БД - на самом деле Auth_Hash
	_, err = h.storage.CreateUser(ctx, req.Login, finalHash, req.AuthSalt, req.EncryptionSalt)
	if err != nil {
		h.logger.Error("failed to create user", zap.Error(err), zap.String("login", req.Login))
		return nil, status.Error(codes.Internal, "failed to create account")
	}

	h.logger.Info("new user registered", zap.String("login", req.Login))

	return &pb.RegisterResponse{
		Success: true,
		Message: "User registered successfully",
	}, nil
}

// Login аутентифицирует пользователя
// Принимает Auth_Key, сравнивает с Auth_Hash и выдаёт JWT-токен + Encryption_Salt
func (h *AuthHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Login == "" ||
		len(req.AuthKey) != crypto.KeyLen {
		return nil, status.Error(codes.InvalidArgument, "login and auth_key required")
	}

	h.logger.Info("Login request received", zap.String("login", req.Login))

	// 1. Ищем пользователя
	user, err := h.storage.GetUserByLogin(ctx, req.Login)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			// Возвращаем общую ошибку, чтобы не выдавать существование пользователя
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		h.logger.Error("login storage error", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	// 2. Проверяем Auth_Key
	// Сличаем Auth_Key пользователя c Auth_Hash в БД используя Server_Pepper
	valid, err := auth.CheckAuthKey(req.AuthKey, h.conf.ServerPepper, user.PasswordHash)
	if err != nil {
		h.logger.Error("crypto check error", zap.Error(err), zap.String("login", req.Login))
		return nil, status.Error(codes.Internal, "authentication internal error")
	}

	if !valid {
		h.logger.Warn("failed login attempt", zap.String("login", req.Login))
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// 3. Генерируем JWT-токен
	tokenString, err := h.tokman.Generate(user.ID.String())
	if err != nil {
		h.logger.Error("token generation failed", zap.Error(err), zap.String("uid", user.ID.String()))
		return nil, status.Error(codes.Internal, "session error")
	}

	h.logger.Debug("user logged in", zap.String("login", req.Login))

	// Возвращаем токен и Encryption_Salt (необходима клиенту для расшифровки Master_Key)
	return &pb.LoginResponse{
		Token:          tokenString,
		EncryptionSalt: user.EncryptionSalt,
	}, nil
}

// Ping - проверка доступности сервера
func (h *AuthHandler) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	// Проверяем здоровье БД
	if err := h.storage.Ping(ctx); err != nil {
		h.logger.Error("Database ping failed", zap.Error(err))
		return nil, status.Error(codes.Unavailable, "database unavailable")
	}

	h.logger.Info("Ping received", zap.String("msg", req.Message))

	return &pb.PingResponse{
		Message: "Pong: " + req.Message,
		Version: "v0.0.1",
	}, nil
}
