package tlsserv

import (
	"os"
	"testing"

	"go.uber.org/zap"
)

func TestTLSInitialize(t *testing.T) {
	// Инициализируем "пустой" логгер, чтобы не засорять вывод теста
	logger := zap.NewNop()

	// TLSInitialize использует autocert.DirCache("certs"), что создаст папку
	// "certs" в текущей директории при инициализации менеджера - мы должны
	// очистить её после теста
	defer func() {
		err := os.RemoveAll("certs")
		if err != nil {
			t.Logf("failed to cleanup certs dir: %v", err)
		}
	}()

	// Вызов тестируемой функции
	creds, err := TLSInitialize(logger)

	// 1. Проверка на ошибки
	if err != nil {
		t.Fatalf("TLSInitialize failed with error: %v", err)
	}

	// 2. Проверка, что creds не nil
	if creds == nil {
		t.Fatal("Expected TransportCredentials, got nil")
	}

	// 3. Проверка метаданных протокола
	// Метод Info() возвращает структуру с полем SecurityProtocol
	info := creds.Info()
	if info.SecurityProtocol != "tls" {
		t.Errorf("Expected SecurityProtocol to be 'tls', got '%s'", info.SecurityProtocol)
	}

	// 4. Проверка реализации интерфейса (Clone)
	// gRPC credentials должны поддерживать клонирование
	cloned := creds.Clone()
	if cloned == nil {
		t.Error("Clone() returned nil, expected valid credentials copy")
	}
}
