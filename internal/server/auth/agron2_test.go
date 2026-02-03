package auth

import (
	"testing"
)

// Константы для тестов
const (
	testPepper = "super-secret-pepper2"
	// testPass   = "my-strong-password3"
)

// TestGenerateSalt проверяет, что соль генерируется нужной длины и она уникальна
func TestGenerateSalt(t *testing.T) {
	salt1, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt вернул ошибку: %v", err)
	}

	if len(salt1) != 16 {
		t.Errorf("Ожидалась длина соли 16 байт, получено: %d", len(salt1))
	}

	salt2, err := GenerateSalt()
	if err != nil {
		t.Fatalf("Второй вызов GenerateSalt вернул ошибку: %v", err)
	}

	// Крайне маловероятно, но проверяем, что соль не повторяется
	if string(salt1) == string(salt2) {
		t.Error("GenerateSalt вернул два одинаковых значения подряд")
	}
}

// // TestHashPassword_Flow проверяет ручной процесс: генерация соли -> хеширование -> проверка
// func TestHashPassword_Flow(t *testing.T) {
// 	salt, err := GenerateSalt()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// 1. Хешируем
// 	hashStr, err := HashPassword(testPass, salt, testPepper)
// 	if err != nil {
// 		t.Fatalf("HashPassword вернул ошибку: %v", err)
// 	}

// 	fmt.Println(hashStr)

// 	// Проверяем формат PHC строки (базовая проверка префикса)
// 	if !strings.HasPrefix(hashStr, "$argon2id$") {
// 		t.Errorf("Хеш должен начинаться с $argon2id$, получено: %s", hashStr)
// 	}

// 	// 2. Проверяем валидный пароль
// 	valid, err := VerifyPassword(byte[](hashStr), testPepper, hashStr)
// 	if err != nil {
// 		t.Errorf("VerifyPassword вернул ошибку на валидных данных: %v", err)
// 	}
// 	if !valid {
// 		t.Error("VerifyPassword вернул false для верного пароля и перца")
// 	}

// 	// 3. Проверяем НЕвалидный пароль
// 	valid, err = VerifyPassword("wrong-password", testPepper, hashStr)
// 	if err != nil {
// 		t.Errorf("Ошибка при проверке неверного пароля: %v", err)
// 	}
// 	if valid {
// 		t.Error("VerifyPassword вернул true для неверного пароля")
// 	}

// 	// 4. Проверяем НЕвалидный перец
// 	valid, err = VerifyPassword(testPass, "wrong-pepper", hashStr)
// 	if err != nil {
// 		t.Errorf("Ошибка при проверке неверного перца: %v", err)
// 	}
// 	if valid {
// 		t.Error("VerifyPassword вернул true для неверного перца")
// 	}
// }

// TestHashAuthKey_Flow проверяет автоматический процесс (соль внутри) для AuthKey
// func TestHashAuthKey_Flow(t *testing.T) {
// 	authKey := []byte("some-api-key-bytes")

// 	// 1. Хешируем (соль генерируется внутри)
// 	hashStr, err := HashAuthKey(authKey, testPepper)
// 	if err != nil {
// 		t.Fatalf("HashAuthKey вернул ошибку: %v", err)
// 	}

// 	// 2. Проверяем валидность
// 	valid, err := CheckAuthKey(authKey, testPepper, hashStr)
// 	if err != nil {
// 		t.Errorf("CheckAuthKey вернул ошибку на валидных данных: %v", err)
// 	}
// 	if !valid {
// 		t.Error("CheckAuthKey вернул false для верного ключа")
// 	}

// 	// 3. Проверка уникальности (соль должна быть разной при каждом вызове)
// 	hashStr2, _ := HashAuthKey(authKey, testPepper)
// 	if hashStr == hashStr2 {
// 		t.Error("HashAuthKey должен генерировать случайную соль каждый раз, но хеши совпали")
// 	}
// }

// TestCheckAuthKey_Malformed проверяет реакцию на битые строки хеша
func TestCheckAuthKey_Malformed(t *testing.T) {
	cases := []struct {
		name      string
		hash      string
		shouldErr bool
	}{
		{
			name:      "Empty string",
			hash:      "",
			shouldErr: true,
		},
		{
			name:      "Garbage string",
			hash:      "not-a-valid-argon-hash",
			shouldErr: true,
		},
		{
			name:      "Wrong prefix",
			hash:      "$argon2i$v=19$m=65536,t=1,p=2$c2FsdA$hash", // argon2i вместо argon2id
			shouldErr: true,
		},
		{
			name: "Invalid Base64 Salt",
			// Валидный формат, но соль содержит недопустимые символы для base64
			hash:      "$argon2id$v=19$m=65536,t=1,p=2$invalid_b64!$hash",
			shouldErr: true, // scanf может пройти, но decode base64 упадет
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			valid, err := CheckAuthKey([]byte("any"), testPepper, tc.hash)

			// Ожидаем ошибку парсинга
			if tc.shouldErr && err == nil {
				t.Error("Ожидалась ошибка парсинга, но err == nil")
			}

			// В любом случае результат должен быть false
			if valid {
				t.Error("CheckAuthKey вернул true для некорректного хеша")
			}
		})
	}
}
