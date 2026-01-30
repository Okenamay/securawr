package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
)

// EncryptAES шифрует plainText ключом key используя AES-256-GCM
//
// Аргументы:
// - key: 32 байта для AES-256
// - plainText: данные для шифрования
//
// Возвращает:
// - []byte: объединенный массив [Nonce (12 байт) + Ciphertext + Tag (16 байт)]
// - error: если произошла ошибка генерации Nonce или создания блочного шифра
func EncryptAES(key, plainText []byte) ([]byte, error) {
	// 1. Создаем блочный шифр AES
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	// 2. Оборачиваем в GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// 3. Генерируем уникальный Nonce
	// Для GCM стандартный размер Nonce — 12 байт, Nonce никогда не должен
	// повторяться для одного и того же ключа
	nonce, err := GenerateRandomBytes(gcm.NonceSize())
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 4. Шифруем (Seal)
	// gcm.Seal(dst, nonce, plaintext, additionalData)
	// Передаем nonce как dst (первый аргумент), чтобы функция добавила
	// ciphertext прямо к nonce. Результат будет: [Nonce | Ciphertext | Tag]
	ciphertext := gcm.Seal(nonce, nonce, plainText, nil)

	return ciphertext, nil
}

// DecryptAES расшифровывает data ключом key
// Ожидает формат данных: [Nonce (12 байт) + Ciphertext + Tag]
func DecryptAES(key, data []byte) ([]byte, error) {
	// 1. Создаем блочный шифр AES
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	// 2. Разделяем Nonce и сам шифротекст
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	// 3. Расшифровываем (Open)
	// gcm.Open(dst, nonce, ciphertext, additionalData)
	// Проверяет целостность (MAC/Tag) автоматически
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Обычно ошибка: "cipher: message authentication failed"
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}
