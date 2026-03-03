package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/zalando/go-keyring"
	"go.etcd.io/bbolt"
)

const (
	dirName     = ".securawr"
	dbName      = "storage.db"
	bucketName  = "files"        // Имя bucket (таблицы) для файлов
	serviceName = "SecurawrApp"  // Имя сервиса для Keyring
	userKey     = "current_user" // Ключ пользователя для Keyring
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
	// Поля для хранения контента и LRU
	EncryptedData []byte    `json:"encrypted_data"`
	EncryptedKey  []byte    `json:"encrypted_key"`
	LastAccess    time.Time `json:"last_access"`
}

// Storage обертка над BoltDB
type Storage struct {
	db      *bbolt.DB
	dbquota int64 // Лимит хранилища в байтах (0 = безлимит)
}

// NewStorage инициализирует локальное хранилище. Открывает (или создает) файл
func NewStorage(path string, dbquota int64) (*Storage, error) {
	// Если путь передан, используем его
	dbPath := path
	if dbPath == "" {
		// Fallback на дефолтный путь в домашней директории
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home dir: %w", err)
		}

		storageDir := filepath.Join(home, dirName)
		if err := os.MkdirAll(storageDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create storage dir: %w", err)
		}
		dbPath = filepath.Join(storageDir, dbName)
	} else {
		// Создаем директорию, если передан кастомный путь
		if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
			return nil, fmt.Errorf("failed to create storage dir: %w", err)
		}
	}

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

	return &Storage{db: db, dbquota: dbquota}, nil
}

// Close закрывает соединение с БД
func (s *Storage) Close() error {
	return s.db.Close()
}

// enforceQuota проверяет размер и удаляет старые файлы (LRU)
func (s *Storage) enforceQuota(b *bbolt.Bucket, newSize int64) error {
	var totalSize int64
	var items []LocalRecord

	c := b.Cursor()
	// Считаем общий размер и собираем метаданные для сортировки
	for k, v := c.First(); k != nil; k, v = c.Next() {
		var rec LocalRecord
		// Нам нужно распарсить JSON, чтобы получить LastAccess
		if err := json.Unmarshal(v, &rec); err != nil {
			continue // Игнорируем битые записи
		}
		// Размер записи = ключ + значение
		totalSize += int64(len(k) + len(v))
		items = append(items, rec)
	}

	// Если места хватает, выходим
	if totalSize+newSize <= s.dbquota {
		return nil
	}

	// Сортировка по LastAccess (самые старые первыми)
	sort.Slice(items, func(i, j int) bool {
		return items[i].LastAccess.Before(items[j].LastAccess)
	})

	// Удаляем, пока не освободим место
	for _, item := range items {
		if totalSize+newSize <= s.dbquota {
			break
		}

		key := []byte(item.ID)
		val := b.Get(key)
		removedSize := int64(len(key) + len(val))

		if err := b.Delete(key); err != nil {
			return fmt.Errorf("failed to delete evicted item: %w", err)
		}
		totalSize -= removedSize
	}

	if totalSize+newSize > s.dbquota {
		return fmt.Errorf("quota exceeded even after eviction")
	}

	return nil
}

// Save сохраняет или обновляет запись
func (s *Storage) Save(record LocalRecord) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		// Обновляем время доступа при сохранении
		record.LastAccess = time.Now()

		data, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("json marshal error: %w", err)
		}

		// Проверка квоты перед записью
		if s.dbquota > 0 {
			// Размер новой записи
			newSize := int64(len(data))
			if err := s.enforceQuota(b, newSize); err != nil {
				return err
			}
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
				// Если одна запись битая, вернем ошибку
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

	// Update, чтобы обновить LastAccess (LRU)
	err := s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		v := b.Get([]byte(id))
		if v == nil {
			return fmt.Errorf("record not found")
		}
		if err := json.Unmarshal(v, &rec); err != nil {
			return err
		}

		// Обновляем LastAccess
		rec.LastAccess = time.Now()
		newData, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("failed to update access time: %w", err)
		}
		return b.Put([]byte(id), newData)
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

// SaveToken сохраняет токен авторизации в системное безопасное хранилище
// (Keyring)
func SaveToken(token string) error {
	if token == "" {
		return errors.New("token is empty")
	}

	// Сохраняем в Keyring
	err := keyring.Set(serviceName, userKey, token)
	if err != nil {
		return fmt.Errorf("failed to save token to system keyring: %w", err)
	}

	return nil
}

// GetToken получает токен авторизации из системного безопасного хранилища
func GetToken() (string, error) {
	token, err := keyring.Get(serviceName, userKey)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", fmt.Errorf("auth token not found (please login first)")
		}
		return "", fmt.Errorf("failed to retrieve token from system keyring: %w", err)
	}

	return token, nil
}
