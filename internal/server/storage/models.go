package storage

import (
	"time"

	"github.com/google/uuid"
)

// User - структура пользователя, соответствующая схеме БД
type User struct {
	ID           uuid.UUID `db:"id"`
	Login        string    `db:"login"`
	PasswordHash string    `db:"password_hash"`
	Salt         string    `db:"salt"` // Auth_Salt
	// EncryptionSalt []byte    `db:"encryption_salt"`
	// AuthHash       string    `db:"auth_hash"`
	// PepperVersion  int       `db:"pepper_version"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// DataRecord - структура, представляющая собой единицу хранимых данных (файл,
// текст, данные карты и т.п.)
type DataRecord struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	DataType  int       `db:"data_type"` // proto.DataType
	DataBlob  []byte    `db:"data_blob"` // Сам контент
	MetaInfo  string    `db:"meta_info"` // Метаданные в JSON-строке
	Version   int       `db:"version"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// DataMeta - вспомогательная структура для парсинга столбца MetaInfo
type DataMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Filename    string `json:"filename"`
	// Здесь могут быть добавлены дополнительные поля
}
