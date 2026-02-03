package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrDataNotFound = errors.New("data not found")
)

// CreateUser создает нового пользователя с солями
func (s *Storage) CreateUser(ctx context.Context, login, passwordHash string, authSalt, encSalt []byte) (*User, error) {
	query := `
		INSERT INTO users (login, password_hash, auth_salt, encryption_salt)
		VALUES ($1, $2, $3, $4)
		RETURNING id, login, password_hash, auth_salt, encryption_salt, created_at, updated_at
	`
	var user User
	err := s.Pool.QueryRow(ctx, query, login, passwordHash, authSalt, encSalt).Scan(
		&user.ID, &user.Login, &user.PasswordHash, &user.AuthSalt, &user.EncryptionSalt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

// GetUserByLogin ищет пользователя по логину
func (s *Storage) GetUserByLogin(ctx context.Context, login string) (*User, error) {
	query := `SELECT id, login, password_hash, auth_salt, encryption_salt, created_at, updated_at FROM users WHERE login = $1 LIMIT 1`
	var user User
	err := s.Pool.QueryRow(ctx, query, login).Scan(
		&user.ID, &user.Login, &user.PasswordHash, &user.AuthSalt, &user.EncryptionSalt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// CreateDataRecord создает запись данных
func (s *Storage) CreateDataRecord(ctx context.Context, r DataRecord) error {
	query := `
		INSERT INTO data_records (id, user_id, data_type, data_blob, meta_info, version)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := s.Pool.Exec(ctx, query, r.ID, r.UserID, r.DataType, r.DataBlob, r.MetaInfo, r.Version)
	if err != nil {
		return fmt.Errorf("failed to create data record: %w", err)
	}
	return nil
}

// ListDataRecords возвращает список записей пользователя (без самих данных)
func (s *Storage) ListDataRecords(ctx context.Context, userID uuid.UUID) ([]DataRecord, error) {
	query := `
		SELECT id, user_id, data_type, data_blob, meta_info, version, created_at, updated_at 
		FROM data_records 
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list data records: %w", err)
	}
	defer rows.Close()

	var records []DataRecord
	for rows.Next() {
		var r DataRecord
		if err := rows.Scan(&r.ID, &r.UserID, &r.DataType, &r.DataBlob, &r.MetaInfo, &r.Version, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan data record: %w", err)
		}
		records = append(records, r)
	}
	return records, nil
}

// GetDataRecord возвращает запись по ID и UserID
func (s *Storage) GetDataRecord(ctx context.Context, id, userID uuid.UUID) (*DataRecord, error) {
	query := `
		SELECT id, user_id, data_type, data_blob, meta_info, version, created_at, updated_at 
		FROM data_records 
		WHERE id = $1 AND user_id = $2
	`
	var r DataRecord
	err := s.Pool.QueryRow(ctx, query, id, userID).Scan(
		&r.ID, &r.UserID, &r.DataType, &r.DataBlob, &r.MetaInfo, &r.Version, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDataNotFound
		}
		return nil, fmt.Errorf("failed to get data record: %w", err)
	}
	return &r, nil
}

// DeleteDataRecord удаляет запись
func (s *Storage) DeleteDataRecord(ctx context.Context, id, userID uuid.UUID) error {
	query := `DELETE FROM data_records WHERE id = $1 AND user_id = $2`
	tag, err := s.Pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete data record: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDataNotFound
	}
	return nil
}

// GetUserUsage возвращает суммарный объем данных пользователя в байтах
func (s *Storage) GetUserUsage(ctx context.Context, userID uuid.UUID) (int64, error) {
	// Считаем сумму длин зашифрованных данных и ключей
	query := `
		SELECT COALESCE(SUM(octet_length(data_blob)), 0)
		FROM data_records
		WHERE user_id = $1
	`
	var usage int64
	err := s.Pool.QueryRow(ctx, query, userID).Scan(&usage)
	if err != nil {
		return 0, fmt.Errorf("failed to get user usage: %w", err)
	}
	return usage, nil
}
