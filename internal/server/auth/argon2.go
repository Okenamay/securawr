package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Параметры Argon2id согласно Дизайн-документу
const (
	ArgonTime    = 3
	ArgonMemory  = 64 * 1024 // 64 MB
	ArgonThreads = 4
	SaltLength   = 16
	KeyLength    = 32
)

// HashPassword создает хеш из ключа (auth_key), соли и серверного перца.
// Использует формат: $argon2id$v=19$m=65536,t=3,p=4$salt$hash
func HashPassword(authKey, salt, pepper string) (string, error) {
	saltBytes, err := base64.RawStdEncoding.DecodeString(salt)
	if err != nil {
		return "", fmt.Errorf("failed to decode salt: %w", err)
	}

	// Смешиваем ключ клиента с серверным перцем перед хешированием
	combinedKey := authKey + pepper

	hash := argon2.IDKey(
		[]byte(combinedKey),
		saltBytes,
		ArgonTime,
		ArgonMemory,
		ArgonThreads,
		KeyLength,
	)

	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Формируем строку для хранения в БД
	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		ArgonMemory,
		ArgonTime,
		ArgonThreads,
		salt,
		b64Hash,
	)

	return encodedHash, nil
}

// VerifyPassword проверяет соответствие ключа сохраненному хешу
func VerifyPassword(authKey, pepper, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}

	// Извлекаем параметры из закодированной строки
	var version, memory, time, threads int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, err
	}
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false, err
	}

	saltBytes, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	combinedKey := authKey + pepper
	comparisonHash := argon2.IDKey(
		[]byte(combinedKey),
		saltBytes,
		uint32(time),
		uint32(memory),
		uint8(threads),
		uint32(len(decodedHash)),
	)

	// Побитовое сравнение для предотвращения атак по времени
	return subtle.ConstantTimeCompare(decodedHash, comparisonHash) == 1, nil
}

// GenerateSalt создает новую случайную соль в base64
func GenerateSalt() (string, error) {
	b := make([]byte, SaltLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(b), nil
}
