package crypto

import (
	"fmt"
)

// Seal (Запечатать) реализует схему Envelope Encryption
//
// Алгоритм:
// 1. Генерирует новый уникальный ключ данных (DEK, 32 байта);
// 2. Шифрует payload с помощью DEK;
// 3. Шифрует сам DEK с помощью masterKey;
// 4. Возвращает объект, готовый к сохранению
//
// masterKey должен быть валидным ключом AES-256 (32 байта)
func Seal(masterKey, payload []byte) (*EncryptedObject, error) {
	// 1. Генерируем случайный DEK (Data Encryption Key)
	dek, err := GenerateRandomBytes(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate DEK: %w", err)
	}

	// 2. Шифруем данные с помощью DEK
	encryptedData, err := EncryptAES(dek, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt payload: %w", err)
	}

	// 3. Шифруем DEK с помощью Master Key
	encryptedDEK, err := EncryptAES(masterKey, dek)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt DEK: %w", err)
	}

	return &EncryptedObject{
		EncryptedData: encryptedData,
		EncryptedDEK:  encryptedDEK,
	}, nil
}

// Open (Открыть) расшифровывает EncryptedObject
//
// Алгоритм:
// 1. Расшифровывает DEK с помощью masterKey;
// 2. Используя полученный DEK, расшифровывает данные
func Open(masterKey []byte, obj *EncryptedObject) ([]byte, error) {
	if obj == nil {
		return nil, fmt.Errorf("encrypted object is nil")
	}

	// 1. Достаем DEK (Data Encryption Key)
	dek, err := DecryptAES(masterKey, obj.EncryptedDEK)
	if err != nil {
		// Частая ошибка: если masterKey неверен, GCM вернет ошибку
		// аутентификации
		return nil, fmt.Errorf("failed to decrypt DEK (invalid master key?): %w", err)
	}

	// 2. Расшифровываем сами данные
	data, err := DecryptAES(dek, obj.EncryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data content: %w", err)
	}

	return data, nil
}
