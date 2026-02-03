package crypto

import (
	"bytes"
	"testing"
)

func TestEnvelope_SealOpen(t *testing.T) {
	// 1. Подготовка Master Key (как будто получен из KDF)
	masterKey, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	originalData := []byte("My secret credit card number: 1234-5678-9012-3456")

	// 2. Seal (Запечатываем)
	envelope, err := Seal(masterKey, originalData)
	if err != nil {
		t.Fatalf("Seal failed: %v", err)
	}

	// Проверки структуры
	if len(envelope.EncryptedData) == 0 {
		t.Error("EncryptedData is empty")
	}
	if len(envelope.EncryptedDEK) == 0 {
		t.Error("EncryptedDEK is empty")
	}
	if bytes.Equal(envelope.EncryptedData, originalData) {
		t.Error("EncryptedData matches original data (no encryption?)")
	}

	// 3. Open (Открываем правильным ключом)
	restoredData, err := Open(masterKey, envelope)
	if err != nil {
		t.Fatalf("Open failed with correct key: %v", err)
	}

	if !bytes.Equal(originalData, restoredData) {
		t.Errorf("Restored data mismatch.\nWant: %s\nGot: %s", originalData, restoredData)
	}
}

func TestEnvelope_WrongMasterKey(t *testing.T) {
	masterKey, _ := GenerateRandomBytes(32)
	wrongKey, _ := GenerateRandomBytes(32)

	data := []byte("secret")
	envelope, _ := Seal(masterKey, data)

	// Пытаемся открыть чужим ключом
	_, err := Open(wrongKey, envelope)
	if err == nil {
		t.Error("Expected error when opening with wrong master key, got nil")
	}
}

func TestEnvelope_TamperedDEK(t *testing.T) {
	masterKey, _ := GenerateRandomBytes(32)
	data := []byte("secret")
	envelope, _ := Seal(masterKey, data)

	// Портим зашифрованный DEK
	envelope.EncryptedDEK[0] ^= 0xFF

	_, err := Open(masterKey, envelope)
	if err == nil {
		t.Error("Expected error when DEK is tampered, got nil")
	}
}

func TestEnvelope_TamperedData(t *testing.T) {
	masterKey, _ := GenerateRandomBytes(32)
	data := []byte("secret")
	envelope, _ := Seal(masterKey, data)

	// Портим сами данные, но DEK оставляем целым
	// DecryptAES должен заметить подмену тега данных
	envelope.EncryptedData[0] ^= 0xFF

	_, err := Open(masterKey, envelope)
	if err == nil {
		t.Error("Expected error when Data is tampered, got nil")
	}
}
