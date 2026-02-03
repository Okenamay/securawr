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

	// Тесты на битые хеши
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
