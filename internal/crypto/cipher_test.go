package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptAES(t *testing.T) {
	// 32-байтный ключ (AES-256)
	key, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	originalText := []byte("Hello, SecuRawr! This is a secret message.")

	// 1. Шифрование
	encrypted, err := EncryptAES(key, originalText)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	if bytes.Equal(originalText, encrypted) {
		t.Fatal("Encrypted data matches original (plaintext leaked?)")
	}

	// 2. Дешифрование
	decrypted, err := DecryptAES(key, encrypted)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	// 3. Сравнение
	if !bytes.Equal(originalText, decrypted) {
		t.Errorf("Decrypted text does not match original.\nGot: %s\nWant: %s", decrypted, originalText)
	}
}

func TestDecryptAES_WrongKey(t *testing.T) {
	key1, _ := GenerateRandomBytes(32)
	key2, _ := GenerateRandomBytes(32) // Другой ключ

	text := []byte("Sensitive Data")
	encrypted, _ := EncryptAES(key1, text)

	// Пытаемся расшифровать неверным ключом
	_, err := DecryptAES(key2, encrypted)
	if err == nil {
		t.Error("Expected error when decrypting with wrong key, got nil")
	}
}

func TestDecryptAES_TamperedData(t *testing.T) {
	key, _ := GenerateRandomBytes(32)
	text := []byte("Sensitive Data")
	encrypted, _ := EncryptAES(key, text)

	// Изменяем последний байт (часть тега или данных)
	encrypted[len(encrypted)-1] ^= 0xFF

	_, err := DecryptAES(key, encrypted)
	if err == nil {
		t.Error("Expected error for tampered data, got nil")
	}
}

func TestEncryptAES_KeySize(t *testing.T) {
	// Неверный размер ключа (10 байт вместо 16/24/32)
	shortKey := []byte("shortkey")
	_, err := EncryptAES(shortKey, []byte("data"))
	if err == nil {
		t.Error("Expected error for invalid key size, got nil")
	}
}
