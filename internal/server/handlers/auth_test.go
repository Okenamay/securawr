package handlers

import (
	"context"
	"testing"
	"time"

	pb "github.com/Okenamay/securawr/gen/proto" // Предполагаем наличие констант тут
	"github.com/Okenamay/securawr/internal/crypto"
	"github.com/Okenamay/securawr/internal/server/auth/token"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAuthHandler_RegisterAndLogin(t *testing.T) {
	store, cfg, logger, teardown := setupTestEnv(t)
	defer teardown()

	tokMan := token.New("test_jwt_secret", time.Hour)
	handler := NewAuthHandler(store, cfg, logger, tokMan)
	ctx := context.Background()

	login := "test_user"
	// Генерируем фейковые ключи нужной длины
	authKey := make([]byte, crypto.KeyLen)
	authSalt := make([]byte, crypto.SaltLen)
	encSalt := make([]byte, crypto.SaltLen)

	// 1. Register
	reqReg := &pb.RegisterRequest{
		Login:          login,
		AuthKey:        authKey,
		AuthSalt:       authSalt,
		EncryptionSalt: encSalt,
	}

	_, err := handler.Register(ctx, reqReg)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Попытка повторной регистрации (должна вернуть AlreadyExists)
	_, err = handler.Register(ctx, reqReg)
	if status.Code(err) != codes.AlreadyExists {
		t.Errorf("Expected AlreadyExists for duplicate user, got: %v", err)
	}

	// 2. Login
	reqLogin := &pb.LoginRequest{
		Login:   login,
		AuthKey: authKey, // Тот же ключ
	}

	respLogin, err := handler.Login(ctx, reqLogin)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if respLogin.Token == "" {
		t.Error("Expected JWT token, got empty string")
	}
	if string(respLogin.EncryptionSalt) != string(encSalt) {
		t.Error("EncryptionSalt mismatch in login response")
	}

	// 3. Login with wrong key
	wrongKey := make([]byte, 32)
	wrongKey[0] = 1 // Изменяем байт (делаем ключ неверным)
	reqLoginWrong := &pb.LoginRequest{
		Login:   login,
		AuthKey: wrongKey,
	}
	_, err = handler.Login(ctx, reqLoginWrong)
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("Expected Unauthenticated for wrong key, got: %v", err)
	}
}

func TestAuthHandler_GetAuthParams(t *testing.T) {
	store, cfg, logger, teardown := setupTestEnv(t)
	defer teardown()

	handler := NewAuthHandler(store, cfg, logger, nil)
	ctx := context.Background()

	// 1. Пользователь не существует (должен вернуть фейковую соль, но не ошибку)
	req := &pb.AuthParamsRequest{Login: "unknown_user"}
	resp, err := handler.GetAuthParams(ctx, req)
	if err != nil {
		t.Fatalf("GetAuthParams failed for unknown user: %v", err)
	}
	if len(resp.AuthSalt) != 16 {
		t.Errorf("Expected 16 bytes fake salt, got %d", len(resp.AuthSalt))
	}

	// 2. Существующий пользователь
	// Сначала создадим его напрямую через Storage (для надежности)
	realSalt := []byte("1234567890123456")
	_, err = store.CreateUser(ctx, "exist_user", "hash", realSalt, []byte("enc_salt"))
	if err != nil {
		t.Fatalf("Failed to prepare user: %v", err)
	}

	reqExist := &pb.AuthParamsRequest{Login: "exist_user"}
	respExist, err := handler.GetAuthParams(ctx, reqExist)
	if err != nil {
		t.Fatalf("GetAuthParams failed for existing user: %v", err)
	}
	if string(respExist.AuthSalt) != string(realSalt) {
		t.Errorf("Expected real salt %v, got %v", realSalt, respExist.AuthSalt)
	}
}

// TestAuthHandler_Errors проверяет граничные случаи и ошибки для увеличения
// покрытия (coverage)
func TestAuthHandler_Errors(t *testing.T) {
	store, cfg, logger, teardown := setupTestEnv(t)
	defer teardown()

	tokMan := token.New("secret", time.Hour)
	handler := NewAuthHandler(store, cfg, logger, tokMan)
	ctx := context.Background()

	// Подготовка данных
	login := "error_test_user"
	authKey := make([]byte, 32)
	authSalt := make([]byte, 16)
	encSalt := make([]byte, 16)

	// 1. Успешная регистрация
	_, err := handler.Register(ctx, &pb.RegisterRequest{
		Login:          login,
		AuthKey:        authKey,
		AuthSalt:       authSalt,
		EncryptionSalt: encSalt,
	})
	if err != nil {
		t.Fatalf("Setup registration failed: %v", err)
	}

	// 2. Попытка регистрации дубликата (Проверка поля Success в ответе, если
	// бы оно возвращалось, но handler.Register возвращает error для
	// дубликатов, так что проверяем ошибку)
	_, err = handler.Register(ctx, &pb.RegisterRequest{
		Login:          login, // Тот же логин
		AuthKey:        authKey,
		AuthSalt:       authSalt,
		EncryptionSalt: encSalt,
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Errorf("Expected AlreadyExists error for duplicate, got %v", err)
	}

	// 3. Логин несуществующего пользователя
	_, err = handler.Login(ctx, &pb.LoginRequest{
		Login:   "ghost_user",
		AuthKey: authKey,
	})
	// Здесь ожидается Unauthenticated, чтобы не раскрывать наличие
	// пользователя
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("Expected Unauthenticated for non-existent user, got %v", err)
	}

	// 4. Логин с пустыми полями (валидация входных данных)
	_, err = handler.Login(ctx, &pb.LoginRequest{
		Login:   "",
		AuthKey: authKey,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument for empty login, got %v", err)
	}
}
