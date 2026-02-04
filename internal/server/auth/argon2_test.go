package auth

import (
	"strings"
	"testing"
)

func TestGenerateSalt(t *testing.T) {
	salt1, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt failed: %v", err)
	}
	if len(salt1) != 16 {
		t.Errorf("Expected 16 bytes salt, got %d", len(salt1))
	}

	salt2, _ := GenerateSalt()
	if string(salt1) == string(salt2) {
		t.Error("GenerateSalt produced duplicate values (highly unlikely)")
	}
}

func TestHashAndVerifyPassword(t *testing.T) {
	password := "super_secure_pass"
	pepper := "secret_pepper"

	// 1. Генерируем хеш
	salt, _ := GenerateSalt()
	hash, err := HashPassword(password, salt, pepper)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Проверяем формат PHC строки
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("Hash format seems wrong: %s", hash)
	}

	// 2. Проверяем валидность (успешный кейс)
	valid, err := VerifyPassword(password, pepper, hash)
	if err != nil {
		t.Fatalf("VerifyPassword error: %v", err)
	}
	if !valid {
		t.Error("VerifyPassword returned false for valid credentials")
	}

	// 3. Неверный пароль
	valid, _ = VerifyPassword("wrong_pass", pepper, hash)
	if valid {
		t.Error("VerifyPassword returned true for wrong password")
	}

	// 4. Неверный перец
	valid, _ = VerifyPassword(password, "wrong_pepper", hash)
	if valid {
		t.Error("VerifyPassword returned true for wrong pepper")
	}
}

func TestCheckAuthKey_InvalidInputs(t *testing.T) {
	pepper := "p"
	key := []byte("key")

	// Базовые тесты на битые хеши
	tests := []struct {
		name string
		hash string
	}{
		{"Empty", ""},
		{"Garbage", "not_a_hash"},
		{"Wrong prefix", "$bcrypt$v=1..."},
		{"Invalid params", "$argon2id$v=A$m=B..."},
		{"Bad Base64 Salt", "$argon2id$v=19$m=1,t=1,p=1$!!!bad_base64!!!$hash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := CheckAuthKey(key, pepper, tt.hash)
			if valid {
				t.Error("Expected valid=false")
			}
			if err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

// TestCheckAuthKey_DetailedErrors проверяет конкретные сообщения об ошибках
// для повышения покрытия (coverage) валидации формата хеша.
func TestCheckAuthKey_DetailedErrors(t *testing.T) {
	key := []byte("key")
	pepper := "p"

	tests := []struct {
		name      string
		hash      string
		wantError string // Часть текста ошибки, которую мы ожидаем
	}{
		{
			"Invalid parts count",
			"$argon2id$v=19$m=64", // Слишком мало частей (меньше 6)
			"invalid hash format",
		},
		{
			"Wrong variant",
			"$bcrypt$v=19$m=1,t=1,p=1$salt$hash", // Не argon2id
			"incompatible variant",
		},
		{
			"Wrong version",
			"$argon2id$v=99$m=1,t=1,p=1$salt$hash", // Неподдерживаемая версия
			"incompatible version",
		},
		{
			"Bad params format",
			"$argon2id$v=19$m=BAD,t=1,p=1$salt$hash", // Ошибка парсинга чисел
			"failed to parse params",
		},
		{
			"Bad salt encoding",
			"$argon2id$v=19$m=65536,t=1,p=2$NOT_BASE64$hash", // Соль не base64
			"invalid salt encoding",
		},
		{
			"Bad hash encoding",
			"$argon2id$v=19$m=65536,t=1,p=2$c2FsdA$NOT_BASE64", // Хеш не base64 (c2FsdA = "salt")
			"invalid hash encoding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := CheckAuthKey(key, pepper, tt.hash)
			if valid {
				t.Error("Expected valid=false")
			}
			if err == nil {
				t.Error("Expected error, got nil")
			} else if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("Error %q does not contain expected substring %q", err.Error(), tt.wantError)
			}
		})
	}
}
