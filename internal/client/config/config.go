package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	dirName        = ".securawr"
	configFileName = "config.json"

	// DefaultServerAddress - адрес сервера по умолчанию
	DefaultServerAddress = "localhost:3200"
)

// Config описывает структуру конфигурационного файла
type Config struct {
	ServerAddress  string `json:"server_address"`
	AuthToken      string `json:"auth_token,omitempty"`
	EncryptionSalt string `json:"encryption_salt,omitempty"`
	CertFile       string `json:"cert_file,omitempty"`
}

// Manager управляет загрузкой и сохранением конфигурации
type Manager struct {
	mu       sync.RWMutex
	filePath string
	cfg      *Config
}

// New создает новый экземпляр менеджера. Он автоматически определяет домашнюю
// директорию пользователя
func New() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home dir: %w", err)
	}

	configDir := filepath.Join(home, dirName)
	// Создаем директорию конфига, если её нет (права 700 - только для
	// владельца)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config dir: %w", err)
	}

	return &Manager{
		filePath: filepath.Join(configDir, configFileName),
		cfg: &Config{
			ServerAddress: DefaultServerAddress,
		},
	}, nil
}

// Load читает конфигурацию с диска. Если файл не существует, возвращает
// дефолтную конфигурацию без ошибки
func (m *Manager) Load() (*Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	file, err := os.Open(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Файла нет — это нормальная ситуация для первого запуска
			return m.cfg, nil
		}
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(m.cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return m.cfg, nil
}

// Save записывает текущую конфигурацию на диск
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	file, err := os.Create(m.filePath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Читабельное форматирование
	if err := encoder.Encode(m.cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}

// GetServerAddress возвращает адрес сервера
func (m *Manager) GetServerAddress() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.ServerAddress
}

// SetServerAddress обновляет адрес сервера и сохраняет конфиг
func (m *Manager) SetServerAddress(addr string) error {
	m.mu.Lock()
	m.cfg.ServerAddress = addr
	m.mu.Unlock()
	return m.Save()
}

// GetToken возвращает текущий токен
func (m *Manager) GetToken() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.AuthToken
}

// SetToken обновляет токен и сохраняет конфиг
func (m *Manager) SetToken(token string) error {
	m.mu.Lock()
	m.cfg.AuthToken = token
	m.mu.Unlock()
	return m.Save()
}

// GetEncryptionSalt возвращает соль шифрования
func (m *Manager) GetEncryptionSalt() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.EncryptionSalt
}

// SetEncryptionSalt обновляет соль шифрования и сохраняет конфиг
func (m *Manager) SetEncryptionSalt(salt string) error {
	m.mu.Lock()
	m.cfg.EncryptionSalt = salt
	m.mu.Unlock()
	return m.Save()
}

// GetCertFile возвращает путь к сертификату
func (m *Manager) GetCertFile() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.CertFile
}
