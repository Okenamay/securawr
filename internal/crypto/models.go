package crypto

// EncryptedObject представляет собой контейнер с зашифрованными данными.
// Именно эта структура (или её сериализованный вид) будет храниться в базе
// данных и в локальном кеше. Использование JSON-тегов позволяет легко
// сериализовать объект для сохранения в БД (jsonb) или передачи по сети, хотя
// для gRPC будет отдельный маппинг
type EncryptedObject struct {
	// EncryptedData — полезная нагрузка, зашифрованная случайным ключом (DEK)
	// Формат: [Nonce (12 байт) + Ciphertext + Tag (16 байт)]
	EncryptedData []byte `json:"encrypted_data"`

	// EncryptedDEK — ключ шифрования данных (DEK), зашифрованный Мастер-ключом
	// пользователя
	// Формат: [Nonce (12 байт) + Ciphertext + Tag (16 байт)]
	EncryptedDEK []byte `json:"encrypted_dek"`
}
