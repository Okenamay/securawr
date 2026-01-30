package storage

import (
	"time"

	"github.com/google/uuid"
)

// User - структура пользователя, соответствующая схеме БД
type User struct {
	ID             uuid.UUID `db:"id"`
	Login          string    `db:"login"`
	PasswordHash   string    `db:"password_hash"`
	AuthSalt       []byte    `db:"auth_salt"`
	EncryptionSalt []byte    `db:"encryption_salt"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// DataRecord - структура, представляющая собой единицу хранимых данных (файл,
// текст, данные карты и т.п.)
type DataRecord struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	DataType  int       `db:"data_type"`
	DataBlob  []byte    `db:"data_blob"`
	MetaInfo  string    `db:"meta_info"`
	Version   int       `db:"version"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// DataMeta - вспомогательная структура для парсинга поля MetaInfo
type DataMeta struct {
	Filename    string `json:"filename,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mime_type,omitempty"`
	Size        int64  `json:"size,omitempty"`
}
