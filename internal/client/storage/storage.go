package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

const (
	dirName    = ".securawr"
	dbName     = "storage.db"
	bucketName = "files" // Имя bucket (таблицы) для файлов
)

// LocalRecord - структура для хранения метаданных файла локально. Она похожа
// на DataMetadata из proto, но с JSON-тегами для сериализации в BoltDB
type LocalRecord struct {
	ID          string    `json:"id"`
	Type        int32     `json:"type"` // Используем int32 для совместимости с proto enum
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Synced      bool      `json:"synced"` // Флаг: true, если данные уже на сервере
}

// Storage обертка над BoltDB
type Storage struct {
	db *bbolt.DB
}

// New инициализирует локальное хранилище. Открывает (или создает) файл
// ~/.securawr/storage.db
func New() (*Storage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home dir: %w", err)
	}

	storageDir := filepath.Join(home, dirName)
	if err := os.MkdirAll(storageDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage dir: %w", err)
	}

	dbPath := filepath.Join(storageDir, dbName)

	// Открываем базу данных. Если файла нет, он будет создан. Timeout 1s
	// нужен, чтобы не зависнуть, если файл заблокирован другим процессом
	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open boltdb: %w", err)
	}

	// Инициализируем бакет (таблицу)
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	return &Storage{db: db}, nil
}

// Close закрывает соединение с БД
func (s *Storage) Close() error {
	return s.db.Close()
}

// Save сохраняет или обновляет запись
func (s *Storage) Save(record LocalRecord) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		data, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("json marshal error: %w", err)
		}

		// Используем ID как ключ
		return b.Put([]byte(record.ID), data)
	})
}

// List возвращает все записи из хранилища
func (s *Storage) List() ([]LocalRecord, error) {
	var items []LocalRecord

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		// Проходим курсором по всем ключам
		return b.ForEach(func(k, v []byte) error {
			var rec LocalRecord
			if err := json.Unmarshal(v, &rec); err != nil {
				// Если одна запись битая, логируем или игнорируем, но здесь
				// вернем ошибку
				return fmt.Errorf("json unmarshal error for key %s: %w", k, err)
			}
			items = append(items, rec)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}
	return items, nil
}

// Get возвращает запись по ID
func (s *Storage) Get(id string) (*LocalRecord, error) {
	var rec LocalRecord

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		v := b.Get([]byte(id))
		if v == nil {
			return fmt.Errorf("record not found")
		}
		return json.Unmarshal(v, &rec)
	})

	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// Delete удаляет запись по ID
func (s *Storage) Delete(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		return b.Delete([]byte(id))
	})
}
