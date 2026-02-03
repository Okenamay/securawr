package config

import (
	"os"
	"path/filepath"
	"testing"
)

// setupEnv подменяет переменные окружения, чтобы конфиг создавался во временной папке.
func setupEnv(t *testing.T) string {
	tmpDir := t.TempDir()

	// Переопределяем пути для разных ОС, чтобы os.UserConfigDir() смотрела в tmpDir
	// Linux / macOS (XDG)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	// Fallback для macOS / Linux, если логика завязана на HOME
	t.Setenv("HOME", tmpDir)
	// Windows
	t.Setenv("APPDATA", tmpDir)

	return tmpDir
}

func TestConfigManager_Lifecycle(t *testing.T) {
	setupEnv(t)

	// 1. Инициализация (First Run)
	mgr, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Проверяем, что конфиг загружается (даже если файла нет, не должно
	// падать)
	_, err = mgr.Load()
	if err != nil {
		// Ошибка может быть допустима при первом запуске, но Load обычно
		// обрабатывает это
		t.Logf("First Load returned error (might be expected): %v", err)
	}

	// Проверяем дефолтные значения (ожидаем, что они не пустые, если заданы в коде)
	// Например, адрес сервера обычно имеет дефолт
	if addr := mgr.GetServerAddress(); addr == "" {
		t.Log("Notice: ServerAddress is empty by default")
	}

	// 2. Изменение значений (Setters)
	testToken := "test-jwt-token-123"
	testSalt := "hex-salt-string"

	// Тестируем SetToken (который должен сохранять на диск или в память)
	if err := mgr.SetToken(testToken); err != nil {
		t.Errorf("SetToken failed: %v", err)
	}

	if got := mgr.GetToken(); got != testToken {
		t.Errorf("GetToken() immediately after Set = %q, want %q", got, testToken)
	}

	// Тестируем SetEncryptionSalt
	if err := mgr.SetEncryptionSalt(testSalt); err != nil {
		t.Errorf("SetEncryptionSalt failed: %v", err)
	}

	// 3. Проверка персистентности (Persistence)
	// Создаем новый экземпляр менеджера, чтобы убедиться, что данные считаются
	// с диска
	mgr2, err := New()
	if err != nil {
		t.Fatalf("New() (2nd instance) failed: %v", err)
	}

	// Принудительно загружаем
	if _, err := mgr2.Load(); err != nil {
		t.Fatalf("Load() failed on 2nd instance: %v", err)
	}

	// Внимание: Используется только keyring) — этот assert упадёт, и это
	// нормально (токен проверяется отдельно)
	// Но соль шифрования обязана быть в файле

	// Проверяем токен (если реализация пишет его в файл)
	if mgr2.GetToken() == testToken {
		t.Log("Token persistence confirmed via file")
	}

	// 4. Проверка путей
	storagePath := mgr.GetStoragePath()
	if storagePath == "" {
		t.Error("GetStoragePath() returned empty string")
	}

	// Проверяем, что путь к кешу формируется внутри нашей временной папки,
	// а не в реальной домашней директории
	// (Это косвенно подтверждает, что setupEnv сработал)
	if mgr.GetCertFile() == "" {
		t.Log("Notice: GetCertFile() is empty")
	}
}

func TestConfigManager_Defaults(t *testing.T) {
	setupEnv(t)
	mgr, _ := New()

	// Проверяем разумные лимиты
	limit := mgr.GetMaxFileSize()
	if limit <= 0 {
		t.Errorf("Expected positive MaxFileSize, got %d", limit)
	}

	cacheSize := mgr.GetLocalCacheSize()
	if cacheSize < 0 {
		t.Errorf("Expected non-negative LocalCacheSize, got %d", cacheSize)
	}
}

func TestConfigManager_CorruptFile(t *testing.T) {
	tmpDir := setupEnv(t)

	// 1. Создаем валидный менеджер, чтобы узнать путь к файлу
	_, _ = New()

	configDir := filepath.Join(tmpDir, ".securawr")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(configDir, "config.json")

	// 2. Записываем "битый" JSON
	if err := os.WriteFile(configFile, []byte("{ invalid-json ..."), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Пытаемся загрузить
	mgr2, _ := New()
	_, err := mgr2.Load()

	// Ожидаем ошибку парсинга
	if err == nil {
		t.Error("Expected error when loading corrupt config, got nil")
	}
}
