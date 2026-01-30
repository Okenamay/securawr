package crypto

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// Параметры алгоритма Argon2id
const (
	ArgonTime    = 3         // Количество итераций
	ArgonMemory  = 64 * 1024 // Объем памяти в КБ (64 MB)
	ArgonThreads = 4         // Степень параллелизма (количество потоков)
	KeyLen       = 32        // Длина генерируемого ключа (32 байта для AES-256)
	SaltLen      = 16        // Рекомендуемая длина соли (16 байт)
)

// DeriveKey генерирует криптостойкий ключ длиной 32 байта из пароля и соли
// используя алгоритм Argon2id
//
// Используется для получения:
// - Master_Key (из пароля и Encryption_Salt)
// - Auth_Key (из пароля и Auth_Salt)
// - Auth_Hash (из Auth_Key и Server_Pepper)
func DeriveKey(password, salt []byte) ([]byte, error) {
	if len(salt) == 0 {
		return nil, fmt.Errorf("salt cannot be empty")
	}

	key := argon2.IDKey(password, salt, ArgonTime, ArgonMemory, ArgonThreads, KeyLen)

	return key, nil
}

// GenerateRandomBytes генерирует n криптостойких случайных байт
func GenerateRandomBytes(n int) ([]byte, error) {
	if n <= 0 {
		return nil, fmt.Errorf("number of bytes must be positive")
	}

	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return b, nil
}
