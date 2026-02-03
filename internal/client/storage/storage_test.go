package storage

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

// setupTestDB создает временную директорию и инициализирует хранилище
func setupTestDB(t *testing.T, quota int64) (*Storage, func()) {
	// Создаем временную директорию для теста
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_storage.db")

	store, err := NewStorage(dbPath, quota)
	if err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}

	return store, func() {
		store.Close()
	}
}

func TestStorage_CRUD(t *testing.T) {
	store, teardown := setupTestDB(t, 0) // Безлимитная квота
	defer teardown()

	rec := LocalRecord{
		ID:        "rec-1",
		Name:      "test.txt",
		CreatedAt: time.Now(),
		// Важно: BoltDB хранит данные как JSON.
		// Если вы используете reflect.DeepEqual в тестах, учитывайте точность времени.
		// При json unmarshal time.Time может потерять наносекунды.
	}

	// 1. Create (Save)
	err := store.Save(rec)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 2. Read (Get)
	fetched, err := store.Get("rec-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if fetched.Name != rec.Name {
		t.Errorf("Expected name %s, got %s", rec.Name, fetched.Name)
	}
	// Проверяем, что LastAccess обновился (должен быть "сейчас")
	if fetched.LastAccess.IsZero() {
		t.Error("LastAccess was not updated")
	}

	// 3. List
	list, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected 1 item, got %d", len(list))
	}

	// 4. Update
	rec.Name = "updated.txt"
	err = store.Save(rec)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	fetched, _ = store.Get("rec-1")
	if fetched.Name != "updated.txt" {
		t.Error("Update didn't work")
	}

	// 5. Delete
	err = store.Delete("rec-1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err = store.Get("rec-1")
	if err == nil {
		t.Error("Expected error after deletion, got nil")
	}
}

func TestStorage_LRU_Quota(t *testing.T) {
	// Создаем записи определенного размера
	// Пустая LocalRecord в JSON весит около 150-200 байт (зависит от полей).
	// Добавим payload, чтобы управлять размером.
	payloadSize := 1000
	payload := make([]byte, payloadSize)

	// Установим квоту, которой хватит на 2 записи, но не на 3.
	// 1 запись ~= 1300 байт (ключ + json с payload).
	// Квота = 3000 байт.
	quota := int64(3000)

	store, teardown := setupTestDB(t, quota)
	defer teardown()

	// Helper to create record
	createRec := func(id string) LocalRecord {
		return LocalRecord{
			ID:            id,
			EncryptedData: payload,
			LastAccess:    time.Now(),
		}
	}

	// 1. Сохраняем первую запись
	if err := store.Save(createRec("1")); err != nil {
		t.Fatalf("Failed to save rec 1: %v", err)
	}
	// Ждем, чтобы LastAccess различался
	time.Sleep(10 * time.Millisecond)

	// 2. Сохраняем вторую запись
	if err := store.Save(createRec("2")); err != nil {
		t.Fatalf("Failed to save rec 2: %v", err)
	}
	time.Sleep(10 * time.Millisecond)

	// Сейчас в базе ~2600 байт из 3000.
	// Обновим запись 1, чтобы она стала "свежей" (LastAccess)
	// Тогда при вытеснении должна удалиться запись 2 (как самая старая по доступу)?
	// Нет, стоп. Get обновляет LastAccess.
	_, _ = store.Get("1") // Теперь "1" самая свежая
	time.Sleep(10 * time.Millisecond)

	// 3. Пытаемся сохранить третью запись. Места нет.
	// Должна удалиться самая старая (LastAccess).
	// Самая старая сейчас "2" (так как "1" мы только что трогали).
	if err := store.Save(createRec("3")); err != nil {
		t.Fatalf("Failed to save rec 3 (should trigger eviction): %v", err)
	}

	// Проверяем:
	// "1" должна остаться (она свежая)
	if _, err := store.Get("1"); err != nil {
		t.Error("Record 1 was unexpectedly evicted")
	}
	// "3" должна быть (мы её только что добавили)
	if _, err := store.Get("3"); err != nil {
		t.Error("Record 3 was not saved")
	}
	// "2" должна исчезнуть (она была самой старой)
	if _, err := store.Get("2"); err == nil {
		t.Error("Record 2 should have been evicted, but it exists")
	}
}

// TestKeyring_Mocked демонстрирует подход к тестированию keyring.
// Реальный keyring.Set требует системных библиотек, которых может не быть в контейнере.
// Мы используем мок библиотеки keyring, подменяя провайдер.
func TestKeyring_Mocked(t *testing.T) {
	// keyring.MockInit() инициализирует хранилище в памяти
	keyring.MockInit()

	token := "secret_jwt_token"

	// 1. Save
	if err := SaveToken(token); err != nil {
		t.Fatalf("SaveToken failed: %v", err)
	}

	// 2. Get
	got, err := GetToken()
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}

	if got != token {
		t.Errorf("Token mismatch: got %s, want %s", got, token)
	}

	// 3. Error handling (Empty token)
	if err := SaveToken(""); err == nil {
		t.Error("Expected error for empty token")
	}
}
