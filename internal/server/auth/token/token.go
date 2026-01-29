package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Manager управляет жизненным циклом JWT токенов
type Manager struct {
	signingKey []byte
	tokenTTL   time.Duration
}

// UserClaims расширяет стандартные клеймы JWT
// Используем Subject из RegisteredClaims для хранения UserID
type UserClaims struct {
	jwt.RegisteredClaims
}

// New создает новый экземпляр менеджера токенов
// secret - секретный ключ для подписи (из конфига)
// ttl - время жизни токена
func New(secret string, ttl time.Duration) *Manager {
	return &Manager{
		signingKey: []byte(secret),
		tokenTTL:   ttl,
	}
}

// Generate создает подписанный JWT токен для указанного userID
func (m *Manager) Generate(userID string) (string, error) {
	claims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   userID, // Храним UserID в поле Subject
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(m.signingKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// Validate проверяет валидность токена и возвращает UserID (Subject), если
// токен валиден
func (m *Manager) Validate(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, func(t *jwt.Token) (interface{}, error) {
		// Обязательно проверяем метод подписи, чтобы избежать атак с подменой
		// алгоритма (например, на "none")
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.signingKey, nil
	})

	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims.Subject, nil
	}

	return "", fmt.Errorf("invalid token claims")
}
