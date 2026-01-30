package storage

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
	ErrDataNotFound = errors.New("data record not found")
)

// CreateUser сохраняет нового пользователя в БД
func (s *Storage) CreateUser(ctx context.Context, u User) error {
	query := `
		INSERT INTO users (id, login, password_hash, salt, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`
	_, err := s.Pool.Exec(ctx, query, u.ID, u.Login, u.PasswordHash, u.AuthSalt)
	if err != nil {
		s.log.Error("Failed to create user", zap.Error(err), zap.String("login", u.Login))
		return err
	}

	s.log.Info("User created successfully", zap.String("login", u.Login), zap.String("id", u.ID.String()))
	return nil
}

// GetUserByLogin ищет пользователя по его логину
func (s *Storage) GetUserByLogin(ctx context.Context, login string) (*User, error) {
	query := `
		SELECT id, login, password_hash, salt, created_at, updated_at
		FROM users
		WHERE login = $1
	`
	var u User
	err := s.Pool.QueryRow(ctx, query, login).Scan(
		&u.ID, &u.Login, &u.PasswordHash, &u.AuthSalt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return &u, nil
}

// Методы для работы с данными

// CreateDataRecord создает новую запись данных
func (s *Storage) CreateDataRecord(ctx context.Context, r DataRecord) error {
	query := `
		INSERT INTO data_records (id, user_id, data_type, data_blob, meta_info, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 1, NOW(), NOW())
	`
	// MetaInfo уже должна быть сериализована в JSON строку вызывающим кодом
	_, err := s.Pool.Exec(ctx, query, r.ID, r.UserID, r.DataType, r.DataBlob, r.MetaInfo)
	if err != nil {
		s.log.Error("Failed to create data record", zap.Error(err), zap.String("user_id", r.UserID.String()))
		return err
	}
	return nil
}

// ListDataRecords возвращает метаданные всех записей пользователя. Мы не
// выбираем data_blob, чтобы не грузить память контентом файлов
func (s *Storage) ListDataRecords(ctx context.Context, userID uuid.UUID) ([]DataRecord, error) {
	query := `
		SELECT id, user_id, data_type, meta_info, created_at, updated_at
		FROM data_records
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DataRecord
	for rows.Next() {
		var r DataRecord
		// Сканируем только те поля, что запросили
		if err := rows.Scan(&r.ID, &r.UserID, &r.DataType, &r.MetaInfo, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, r)
	}

	return result, nil
}

// GetDataRecord возвращает полную запись (включая контент) по ID и UserID
// Проверка UserID обязательна для безопасности (чтобы нельзя было скачать
// чужой файл по ID)
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
		return nil, ErrDataNotFound
	}
	return &r, nil
}

// DeleteDataRecord удаляет запись, принадлежащую конкретному пользователю
func (s *Storage) DeleteDataRecord(ctx context.Context, id, userID uuid.UUID) error {
	query := `DELETE FROM data_records WHERE id = $1 AND user_id = $2`
	res, err := s.Pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrDataNotFound
	}
	return nil
}
