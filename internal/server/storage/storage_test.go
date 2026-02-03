package storage

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/Okenamay/securawr/internal/server/config"
)

// getTestDSN читает DSN из переменных окружения или использует дефолтный
func getTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/securawr_test?sslmode=disable"
	}
	return dsn
}

// setupTestDB инициализирует хранилище и чистит таблицы
func setupTestDB(t *testing.T) (*Storage, func()) {
	logger := zap.NewNop()
	cfg := &config.Config{
		DSN: getTestDSN(),
	}

	// New() автоматически запустит миграции
	store, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to connect/migrate DB: %v", err)
	}

	// Очистка таблиц перед каждым тестом
	ctx := context.Background()
	_, err = store.Pool.Exec(ctx, "TRUNCATE TABLE data_records, users CASCADE")
	if err != nil {
		t.Fatalf("Failed to clean tables: %v", err)
	}

	return store, func() {
		store.Close()
	}
}

func TestUserFlow(t *testing.T) {
	store, teardown := setupTestDB(t)
	defer teardown()

	ctx := context.Background()
	login := fmt.Sprintf("user_%d", time.Now().UnixNano())

	// 1. Создание пользователя
	// (Параметры authSalt и encSalt соответствуют вашей схеме в dbquery.go)
	user, err := store.CreateUser(ctx, login, "hash_123", []byte("salt1"), []byte("salt2"))
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.ID == uuid.Nil {
		t.Error("User ID should not be nil")
	}

	// 2. Поиск пользователя
	found, err := store.GetUserByLogin(ctx, login)
	if err != nil {
		t.Fatalf("GetUserByLogin failed: %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("ID mismatch: want %s, got %s", user.ID, found.ID)
	}
}

func TestDataRecordFlow(t *testing.T) {
	store, teardown := setupTestDB(t)
	defer teardown()
	ctx := context.Background()

	// Создаем пользователя-владельца
	user, _ := store.CreateUser(ctx, "data_owner", "h", []byte("s"), []byte("s"))

	blob := []byte("secret_content")
	recID := uuid.New()

	rec := DataRecord{
		ID:       recID,
		UserID:   user.ID,
		DataType: 2, // Binary
		DataBlob: blob,
		MetaInfo: `{"name": "file.txt"}`,
		Version:  1,
	}

	// 1. Сохранение данных
	if err := store.CreateDataRecord(ctx, rec); err != nil {
		t.Fatalf("CreateDataRecord failed: %v", err)
	}

	// 2. Получение данных
	saved, err := store.GetDataRecord(ctx, recID, user.ID)
	if err != nil {
		t.Fatalf("GetDataRecord failed: %v", err)
	}
	if string(saved.DataBlob) != string(blob) {
		t.Error("DataBlob mismatch")
	}

	// 3. Проверка Usage (после фикса в dbquery.go)
	usage, err := store.GetUserUsage(ctx, user.ID)
	if err != nil {
		t.Errorf("GetUserUsage failed: %v", err)
	}
	if usage != int64(len(blob)) {
		t.Errorf("Usage mismatch: want %d, got %d", len(blob), usage)
	}

	// 4. Удаление
	if err := store.DeleteDataRecord(ctx, recID, user.ID); err != nil {
		t.Errorf("DeleteDataRecord failed: %v", err)
	}
	_, err = store.GetDataRecord(ctx, recID, user.ID)
	if err != ErrDataNotFound {
		t.Errorf("Expected ErrDataNotFound, got %v", err)
	}
}
