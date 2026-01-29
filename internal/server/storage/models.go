package storage

import (
	"time"
)

// User - структура пользователя, соответствующая схеме БД
type User struct {
	ID             int64     `db:"id"`
	Login          string    `db:"login"`
	AuthSalt       []byte    `db:"auth_salt"`
	EncryptionSalt []byte    `db:"encryption_salt"`
	AuthHash       string    `db:"auth_hash"`
	PepperVersion  int       `db:"pepper_version"`
	CreatedAt      time.Time `db:"created_at"`
}
