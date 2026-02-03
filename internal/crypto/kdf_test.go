package crypto

import (
	"bytes"
	"testing"
)

func TestDeriveKey_Determinism(t *testing.T) {
	password := []byte("strong_password_123")
	salt := []byte("random_salt_16_b") // 16 bytes

	// 1. Генерируем ключ первый раз
	key1, err := DeriveKey(password, salt)
	if err != nil {
		t.Fatalf("DeriveKey failed: %v", err)
	}

	// 2. Генерируем ключ второй раз с теми же параметрами
	key2, err := DeriveKey(password, salt)
	if err != nil {
		t.Fatalf("DeriveKey failed: %v", err)
	}

	// 3. Проверяем, что ключи идентичны
	if !bytes.Equal(key1, key2) {
		t.Errorf("DeriveKey is not deterministic. Key1: %x, Key2: %x", key1, key2)
	}
}

func TestDeriveKey_Uniqueness(t *testing.T) {
	password := []byte("same_password")

	// Разные соли
	salt1 := []byte("salt_number_one!")
	salt2 := []byte("salt_number_two!")

	key1, _ := DeriveKey(password, salt1)
	key2, _ := DeriveKey(password, salt2)

	if bytes.Equal(key1, key2) {
		t.Error("DeriveKey produced same keys for different salts")
	}
}

func TestGenerateRandomBytes(t *testing.T) {
	n := 32
	b1, err := GenerateRandomBytes(n)
	if err != nil {
		t.Fatalf("Failed to generate bytes: %v", err)
	}

	if len(b1) != n {
		t.Errorf("Expected %d bytes, got %d", n, len(b1))
	}

	b2, _ := GenerateRandomBytes(n)

	// Вероятность совпадения двух 32-байтных случайных последовательностей ничтожно мала
	if bytes.Equal(b1, b2) {
		t.Error("GenerateRandomBytes produced duplicate sequences")
	}
}

func TestDeriveKey_EmptySalt(t *testing.T) {
	password := []byte("password")
	salt := []byte{}

	_, err := DeriveKey(password, salt)
	if err == nil {
		t.Error("Expected error for empty salt, got nil")
	}
}

// BenchmarkDeriveKey позволяет оценить время выполнения KDF.
// Запустите с `go test -bench=. ./internal/crypto`
func BenchmarkDeriveKey(b *testing.B) {
	password := []byte("benchmark_pass")
	salt := []byte("benchmark_salt__") // 16 bytes

	for i := 0; i < b.N; i++ {
		_, _ = DeriveKey(password, salt)
	}
}
