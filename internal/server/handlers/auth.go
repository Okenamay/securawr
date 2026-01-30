package handlers

import (
	"context"
	"crypto/rand"
	"errors"

	pb "github.com/Okenamay/securawr/gen/proto"
	"github.com/Okenamay/securawr/internal/server/auth"
	"github.com/Okenamay/securawr/internal/server/auth/token"
	"github.com/Okenamay/securawr/internal/server/config"
	"github.com/Okenamay/securawr/internal/server/storage"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthHandler struct {
	pb.UnimplementedAuthServiceServer
	db           *storage.Storage
	cfg          *config.Config
	logger       *zap.Logger
	tokenManager *token.Manager
}

func NewAuthHandler(db *storage.Storage, cfg *config.Config, logger *zap.Logger, tm *token.Manager) *AuthHandler {
	return &AuthHandler{
		db:           db,
		cfg:          cfg,
		logger:       logger,
		tokenManager: tm,
	}
}

// GetAuthParams возвращает соль для аутентификации.
func (h *AuthHandler) GetAuthParams(ctx context.Context, req *pb.AuthParamsRequest) (*pb.AuthParamsResponse, error) {
	if req.Login == "" {
		return nil, status.Error(codes.InvalidArgument, "login is required")
	}

	user, err := h.db.GetUserByLogin(ctx, req.Login)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			// Защита от User Enumeration: возвращаем фейковую соль
			fakeSalt := make([]byte, 16)
			rand.Read(fakeSalt)
			return &pb.AuthParamsResponse{AuthSalt: fakeSalt}, nil
		}
		h.logger.Error("failed to get user", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.AuthParamsResponse{AuthSalt: user.AuthSalt}, nil
}

// Register регистрирует пользователя.
func (h *AuthHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.Login == "" || len(req.AuthKey) == 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid arguments")
	}

	// Хешируем Auth_Key с серверным Перцем
	finalHash, err := auth.HashAuthKey(req.AuthKey, h.cfg.ServerPepper)
	if err != nil {
		h.logger.Error("hashing failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "crypto error")
	}

	// Создаем пользователя
	_, err = h.db.CreateUser(ctx, req.Login, finalHash, req.AuthSalt, req.EncryptionSalt)
	if err != nil {
		h.logger.Error("failed to create user", zap.Error(err))
		// Детали ошибки скрываем
		return nil, status.Error(codes.Internal, "registration failed")
	}

	return &pb.RegisterResponse{Success: true, Message: "Registered successfully"}, nil
}

// Login проверяет вход.
func (h *AuthHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, err := h.db.GetUserByLogin(ctx, req.Login)
	if err != nil {
		// Всегда возвращаем generic ошибку
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// Проверяем хеш
	valid, err := auth.CheckAuthKey(req.AuthKey, h.cfg.ServerPepper, user.PasswordHash)
	if err != nil {
		h.logger.Error("crypto check error", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	if !valid {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// Генерируем токен
	tokenStr, err := h.tokenManager.NewJWT(user.ID.String())
	if err != nil {
		h.logger.Error("token gen failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.LoginResponse{
		Token:          tokenStr,
		EncryptionSalt: user.EncryptionSalt,
	}, nil
}

func (h *AuthHandler) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Message: "Pong", Version: "0.0.1"}, nil
}
