package config

import (
	"encoding/json"
	"flag"
	"os"
	"strconv"
	"sync"
)

// Дефолтные значения до применения флагов:
const (
	defaultGRPCPort     = ":8080"
	defaultDSN          = "postgres://postgres:postgres@localhost:5432/securawr_test?sslmode=disable"
	defaultLogLevel     = "info"
	defaultServerPepper = "Cayenne&blacK"
	defaultJWTSecret    = "3hvost@kotyaki:Qu#Oli#Ma@2025!"
	defaultUserQuota    = 250 * 1024 * 1024
	defaultMaxFileSize  = 50 * 1024 * 1024
)

// Config хранит конфигурацию сервера
type Config struct {
	GRPCPort     string
	DSN          string
	LogLevel     string
	ConfigPath   string
	ServerPepper string // Секретная строка сервера для хеширования (Server_Pepper)
	JWTSecret    string // Секрет для подписи токенов
	UserQuota    int64  // Максимальный объём хранилища для одного пользователя (в байтах)
	MaxFileSize  int64  // Максимальный размер одного файла (в байтах)
}

// fileConfig описывает структуру JSON-файла конфигурации
type fileConfig struct {
	GRPCPort     *string `json:"grpc_port"`
	DSN          *string `json:"database_dsn"`
	LogLevel     *string `json:"log_level"`
	ServerPepper *string `json:"server_pepper"`
	JWTSecret    *string `json:"jwt_secret"`
	UserQuota    *int64  `json:"user_quota"`
	MaxFileSize  *int64  `json:"max_file_size"`
}

var (
	once    sync.Once
	cfg     *Config
	initErr error
)

// InitConfig инициализирует конфигурацию один раз (Singleton)
func InitConfig() (*Config, error) {
	once.Do(func() {
		cfg, initErr = parseFlags()
	})
	return cfg, initErr
}

// parseFlags загружает конфигурацию из флагов, файла и переменных окружения
// Приоритет: Flag > Env > Config File > Default
func parseFlags() (*Config, error) {
	config := &Config{}

	// 1. Инициализируемся дефолтными значениями
	config.GRPCPort = defaultGRPCPort
	config.DSN = defaultDSN
	config.LogLevel = defaultLogLevel
	config.ServerPepper = defaultServerPepper
	config.JWTSecret = defaultJWTSecret
	config.UserQuota = defaultUserQuota
	config.MaxFileSize = defaultMaxFileSize

	// 2. Определяем флаги
	flag.StringVar(&config.GRPCPort, "port", config.GRPCPort, "Порт сервера gRPC")
	flag.StringVar(&config.DSN, "dsn", config.DSN, "DSN БД PostgreSQL")
	flag.StringVar(&config.LogLevel, "log-level", config.LogLevel, "Уровень лог-файла (debug, info, error)")
	flag.StringVar(&config.ServerPepper, "pepper", config.ServerPepper, "Перец сервера для повышения защищённости данных")
	flag.StringVar(&config.JWTSecret, "jwt-secret", config.JWTSecret, "Секрет для JWT-токена")
	flag.Int64Var(&config.UserQuota, "user-quota", config.UserQuota, "Квота пользователя на сервере (в байтах)")
	flag.Int64Var(&config.MaxFileSize, "max-file-size", config.MaxFileSize, "Максимальный размер файла (в байтах)")

	// Флаги работы через файл конфигурации
	flag.StringVar(&config.ConfigPath, "config", "", "Путь к файлу конфигурации")
	flag.StringVar(&config.ConfigPath, "c", "", "Путь к файлу конфигурации")

	flag.Parse()

	// Собираем информацию о флагах, явно установленных пользователем
	setFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	// Определяем путь к конфигу
	if config.ConfigPath == "" {
		if cfgPathEnv, ok := os.LookupEnv("CONFIG"); ok {
			config.ConfigPath = cfgPathEnv
		}
	}

	// 3. Загружаем значения из файла (если есть), но пока не применяем
	var fCfg *fileConfig
	if config.ConfigPath != "" {
		var err error
		fCfg, err = loadConfigFromFile(config.ConfigPath)
		if err != nil {
			return nil, err
		}
	}

	// 4. Применяем значения с учетом приоритета. Если флаг установлен, он уже
	// в config (шаг 2), и мы ничего не делаем, если нет — смотрим Env. Если
	// нет Env — применяем значения из файла (шаг 3)

	// GRPCPort
	if !setFlags["port"] {
		if envVal, ok := os.LookupEnv("GRPC_PORT"); ok {
			config.GRPCPort = envVal
		} else if fCfg != nil && fCfg.GRPCPort != nil {
			config.GRPCPort = *fCfg.GRPCPort
		}
	}

	// DSN
	if !setFlags["dsn"] {
		if envVal, ok := os.LookupEnv("DATABASE_DSN"); ok {
			config.DSN = envVal
		} else if fCfg != nil && fCfg.DSN != nil {
			config.DSN = *fCfg.DSN
		}
	}

	// LogLevel
	if !setFlags["log-level"] {
		if envVal, ok := os.LookupEnv("LOG_LEVEL"); ok {
			config.LogLevel = envVal
		} else if fCfg != nil && fCfg.LogLevel != nil {
			config.LogLevel = *fCfg.LogLevel
		}
	}

	// ServerPepper
	if !setFlags["pepper"] {
		if envVal, ok := os.LookupEnv("PEPPER"); ok {
			config.ServerPepper = envVal
		} else if fCfg != nil && fCfg.ServerPepper != nil {
			config.ServerPepper = *fCfg.ServerPepper
		}
	}

	// JWTSecret
	if !setFlags["jwt-secret"] {
		if envVal, ok := os.LookupEnv("JWT_SECRET"); ok {
			config.JWTSecret = envVal
		} else if fCfg != nil && fCfg.JWTSecret != nil {
			config.JWTSecret = *fCfg.JWTSecret
		}
	}

	// UserQuota
	if !setFlags["user-quota"] {
		if envVal, ok := os.LookupEnv("USER_QUOTA"); ok {
			if v, err := strconv.ParseInt(envVal, 10, 64); err == nil {
				config.UserQuota = v
			}
		} else if fCfg != nil && fCfg.UserQuota != nil {
			config.UserQuota = *fCfg.UserQuota
		}
	}

	// MaxFileSize
	if !setFlags["max-file-size"] {
		if envVal, ok := os.LookupEnv("MAX_FILE_SIZE"); ok {
			if v, err := strconv.ParseInt(envVal, 10, 64); err == nil {
				config.MaxFileSize = v
			}
		} else if fCfg != nil && fCfg.MaxFileSize != nil {
			config.MaxFileSize = *fCfg.MaxFileSize
		}
	}

	return config, nil
}

func loadConfigFromFile(path string) (*fileConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var fCfg fileConfig
	if err := json.NewDecoder(file).Decode(&fCfg); err != nil {
		return nil, err
	}

	return &fCfg, nil
}
