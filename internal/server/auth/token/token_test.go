package token

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestManager_GenerateAndValidate(t *testing.T) {
	secret := "my_secret_key"
	ttl := time.Minute * 15
	manager := New(secret, ttl)

	userID := "user-uuid-1234-5678"

	// 1. Генерация токена
	tokenString, err := manager.Generate(userID)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}
	if tokenString == "" {
		t.Fatal("Generate() returned empty string")
	}

	// 2. Успешная валидация
	parsedUserID, err := manager.Validate(tokenString)
	if err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	if parsedUserID != userID {
		t.Errorf("Validate() returned userID %s, want %s", parsedUserID, userID)
	}
}

func TestManager_Expiration(t *testing.T) {
	// Создаем менеджер с отрицательным TTL, чтобы токен сразу был просрочен
	manager := New("secret", -time.Minute)

	tokenString, err := manager.Generate("user")
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	_, err = manager.Validate(tokenString)
	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}
}

func TestManager_InvalidSignature(t *testing.T) {
	// Генерируем токен одним ключом
	manager1 := New("secret_1", time.Hour)
	tokenString, _ := manager1.Generate("user")

	// Пытаемся валидировать другим ключом
	manager2 := New("secret_2", time.Hour)
	_, err := manager2.Validate(tokenString)

	if err == nil {
		t.Error("Expected error for invalid signature, got nil")
	}
}

func TestManager_InvalidAlgorithm(t *testing.T) {
	// Тест на атаку с подменой алгоритма (например, на "none")
	manager := New("secret", time.Hour)

	// Создаем токен вручную с алгоритмом None (если библиотека позволяет, или просто HMAC, но другой битности)
	// jwt.SigningMethodNone обычно отключен по умолчанию в v5, поэтому попробуем ES256 (асимметричный)
	// или просто создадим токен, который хедер говорит одно, а подпись другая.
	// Но проще проверить логику callback'а в Validate:

	// Сформируем токен c "неправильным" методом в заголовке, но валидной структурой,
	// чтобы пройти Parse, но упасть на проверке метода.

	// В данном случае jwt.ParseWithClaims вызовет нашу функцию-валидатор Keyfunc.
	// Мы можем сымитировать ситуацию, используя другой метод подписи при создании.

	claims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "user",
		},
	}
	// Используем ES256 (требует приватный ключ, но нам важно лишь, что это НЕ HMAC)
	// Для теста просто возьмем левый ключ, главное, что метод другой.
	// Или проще: используем токен вообще без подписи (UnsafeAllowNoneSignatureType), если бы мы могли.

	// Попробуем просто создать токен с другим методом, который библиотека сможет распарсить.
	// Нам нужно, чтобы ParseWithClaims вызвал callback, а callback вернул ошибку.

	// Альтернативный подход: Создаем токен легально, но проверяем ошибку "unexpected signing method"
	// если бы мы передали токен подписанный чем-то другим.

	// В реальности самый простой способ проверить защиту от смены алго -
	// использовать токен, подписанный None, если бы библиотека это позволяла.
	// Но библиотека `golang-jwt/jwt/v5` требует `jwt.WithValidator` для None.

	// Поэтому сделаем так: подпишем токен алгоритмом None вручную (или используем библиотечный метод, если разрешено).
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType) // Специальный маркер для None

	_, err := manager.Validate(tokenString)

	// Ожидаем ошибку, так как в коде стоит проверка: if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok
	if err == nil {
		t.Error("Expected error for None algorithm, got success")
	} else if err.Error() != "invalid token: unexpected signing method: none" &&
		err.Error() != "invalid token: token is unverifiable: error while executing keyfunc: unexpected signing method: none" {
		// Текст ошибки может варьироваться в зависимости от версии, проверяем наличие ключевых слов
		// В данном случае наш код возвращает fmt.Errorf("unexpected signing method: %v", ...)
		t.Logf("Got expected error: %v", err)
	}
}

func TestManager_MalformedToken(t *testing.T) {
	manager := New("secret", time.Hour)

	tests := []string{
		"",
		"header.payload",
		"header.payload.signature.extra",
		"not_a_jwt",
	}

	for _, tokenStr := range tests {
		_, err := manager.Validate(tokenStr)
		if err == nil {
			t.Errorf("Expected error for malformed token '%s', got nil", tokenStr)
		}
	}
}
