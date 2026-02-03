package handlers

import (
	"context"
	"os"
	"testing"

	"go.uber.org/zap"

	"github.com/Okenamay/securawr/internal/server/config"
	"github.com/Okenamay/securawr/internal/server/storage"
)

// getTestDSN возвращает DSN для тестовой БД.
func getTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/securawr_test?sslmode=disable"
	}
	return dsn
}

// setupTestEnv инициализирует Storage, Logger и Config для тестов.
// Возвращает функцию для очистки ресурсов.
func setupTestEnv(t *testing.T) (*storage.Storage, *config.Config, *zap.Logger, func()) {
	logger := zap.NewNop()
	cfg := &config.Config{
		DSN:          getTestDSN(),
		ServerPepper: "TestPepperSecret",
		UserQuota:    1024 * 1024, // 1 MB для тестов
		MaxFileSize:  512 * 1024,  // 512 KB
	}

	store, err := storage.New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to connect to test DB: %v", err)
	}

	// Очистка таблиц перед тестом
	ctx := context.Background()
	_, err = store.Pool.Exec(ctx, "TRUNCATE TABLE data_records, users CASCADE")
	if err != nil {
		t.Fatalf("Failed to truncate tables: %v", err)
	}

	return store, cfg, logger, func() {
		store.Close()
	}
}
