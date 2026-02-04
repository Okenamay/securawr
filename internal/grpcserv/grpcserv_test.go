package grpcserv

import (
	"context"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/Okenamay/securawr/gen/proto"
	"github.com/Okenamay/securawr/internal/server/config"
	"github.com/Okenamay/securawr/internal/server/storage"
)

// getTestDSN дублирует логику получения DSN для тестов
func getTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/securawr_test?sslmode=disable"
	}
	return dsn
}

// setupTestEnv подготавливает БД для теста
func setupTestEnv(t *testing.T) (*storage.Storage, func()) {
	logger := zap.NewNop()
	cfg := &config.Config{
		DSN: getTestDSN(),
	}

	// Инициализируем Storage (это также накатит миграции)
	store, err := storage.New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to connect to test DB: %v", err)
	}

	// Очистка таблиц не обязательна для теста запуска сервера,
	// но полезна, если мы будем проверять логику.
	ctx := context.Background()
	_, _ = store.Pool.Exec(ctx, "TRUNCATE TABLE data_records, users CASCADE")

	return store, func() {
		store.Close()
	}
}

func TestServer_Lifecycle(t *testing.T) {
	// 1. Подготовка зависимостей
	store, teardownDB := setupTestEnv(t)
	defer teardownDB()

	logger := zap.NewNop()

	// Конфигурация с портом :0 позволяет ОС выбрать любой свободный порт
	conf := &config.Config{
		GRPCPort:     ":0",
		JWTSecret:    "test_secret",
		MaxFileSize:  1024 * 1024,
		ServerPepper: "pepper",
	}

	// 2. Создание сервера (передаем nil вместо creds, чтобы использовать insecure режим для теста)
	server, err := New(conf, logger, nil, store)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// 3. Запуск сервера в отдельной горутине (так как Start блокирует выполнение)
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start()
	}()

	// Даем серверу немного времени на инициализацию listener'а
	// В реальных условиях можно использовать retry, но для unit-теста sleep обычно достаточно
	time.Sleep(100 * time.Millisecond)

	// Проверяем, что listener инициализировался
	if server.listener == nil {
		t.Fatal("Server listener is nil, server probably failed to start")
	}

	// Получаем реальный адрес, который выдала ОС
	realAddr := server.listener.Addr().String()
	t.Logf("Server started on %s", realAddr)

	// 4. Подключаем тестового клиента
	conn, err := grpc.NewClient(
		realAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()

	client := pb.NewAuthServiceClient(conn)

	// 5. Выполняем Smoke-тест (вызов Ping)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pingResp, err := client.Ping(ctx, &pb.PingRequest{Message: "Integration Test"})
	if err != nil {
		t.Fatalf("Ping call failed: %v", err)
	}

	expected := "Pong: Integration Test"
	if pingResp.Message != expected {
		t.Errorf("Ping response mismatch: want %q, got %q", expected, pingResp.Message)
	}

	// 6. Остановка сервера
	server.Stop()

	// Ожидаем завершения горутины Start
	select {
	case err := <-errCh:
		// grpc.Serve возвращает nil при успешном GracefulStop (или ошибку, если упал)
		if err != nil {
			t.Logf("Server stopped with error (might be expected): %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Server did not stop in time")
	}
}

// TestServer_New_Validation проверяет валидацию при создании
func TestServer_New(t *testing.T) {
	store, teardown := setupTestEnv(t)
	defer teardown()
	logger := zap.NewNop()
	conf := &config.Config{
		JWTSecret:   "secret",
		MaxFileSize: 100,
	}

	srv, err := New(conf, logger, nil, store)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	if srv == nil {
		t.Fatal("New() returned nil server")
	}
}
