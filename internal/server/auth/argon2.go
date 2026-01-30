package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const (
	ArgonTime    = 1
	ArgonMemory  = 64 * 1024
	ArgonThreads = 2
	KeyLen       = 32
)

// HashAuthKey принимает Auth_Key (полученный от клиента) и Server_Pepper
// Возвращает хеш в формате PHC для сохранения в БД
func HashAuthKey(authKey []byte, pepper string) (string, error) {
	// 1. Генерируем случайную соль для этого хеша (хранится в поле
	// password_hash)
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate random salt: %w", err)
	}

	// 2. Смешиваем Auth_Key и Pepper
	inputMaterial := append(authKey, []byte(pepper)...)

	// 3. Хешируем
	hash := argon2.IDKey(inputMaterial, salt, ArgonTime, ArgonMemory, ArgonThreads, KeyLen)

	// 4. Формируем строку: $argon2id$v=...$m=...$salt$hash
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, ArgonMemory, ArgonTime, ArgonThreads, b64Salt, b64Hash,
	)

	return encodedHash, nil
}

// CheckAuthKey проверяет валидность ключа.
func CheckAuthKey(authKey []byte, pepper string, storedHash string) (bool, error) {
	var version, memory, time, threads int
	var b64Salt, b64Hash string

	// Парсим строку хеша
	_, err := fmt.Sscanf(storedHash, "$argon2id$v=%d$m=%d,t=%d,p=%d$%[^$]$%s",
		&version, &memory, &time, &threads, &b64Salt, &b64Hash)
	if err != nil {
		return false, fmt.Errorf("invalid hash format: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(b64Salt)
	if err != nil {
		return false, fmt.Errorf("invalid salt encoding: %w", err)
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(b64Hash)
	if err != nil {
		return false, fmt.Errorf("invalid hash encoding: %w", err)
	}

	// Повторяем хеширование
	inputMaterial := append(authKey, []byte(pepper)...)
	calculatedHash := argon2.IDKey(inputMaterial, salt, uint32(time), uint32(memory), uint8(threads), uint32(len(decodedHash)))

	// Сравнение за константное время
	if subtle.ConstantTimeCompare(decodedHash, calculatedHash) == 1 {
		return true, nil
	}

	return false, nil
}
