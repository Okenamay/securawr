package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/Okenamay/securawr/internal/client/config"
	"github.com/Okenamay/securawr/internal/client/storage"
	logger "github.com/Okenamay/securawr/internal/logger/zap"
)

var (
	// Глобальные переменные для доступа из других команд пакета cli
	ConfigManager *config.Manager
	LocalStorage  *storage.Storage
	log           *zap.Logger
)

// rootCmd представляет базовую команду (securawr)
var rootCmd = &cobra.Command{
	Use:   "securawr",
	Short: "SecuRawr CLI client",
	Long:  `Secure client for credit cards, passwords and binary files.`,

	// PersistentPreRunE выполняется перед любой командой (login, add и т.д.)
	// Здесь мы инициализируем подключение к БД и загружаем конфиг.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initApp()
	},

	// PersistentPostRunE выполняется перед завершения любой команды
	// Гарантирует, что файл БД будет корректно закрыт (освобождён лок)
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if LocalStorage != nil {
			return LocalStorage.Close()
		}
		return nil
	},
}

// Execute - точка входа для CLI
func Execute() {
	// Инициализация логгера
	var err error
	log, err = logger.New("info")
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	if err := rootCmd.Execute(); err != nil {
		// Cobra сама выводит ошибку
		os.Exit(1)
	}
}

func initApp() error {
	var err error

	// 1. Инициализация менеджера конфигурации
	ConfigManager, err = config.New()
	if err != nil {
		return fmt.Errorf("failed to init config manager: %w", err)
	}

	// Пытаемся загрузить конфиг. Ошибку здесь не считаем фатальной (например,
	// файл конфига еще не создан), но если она есть (битый JSON), Load вернет
	// ошибку, которую можно обработать
	if _, err := ConfigManager.Load(); err != nil {
	}

	// Загрузка токена из Keyring
	// Пытаемся найти сохраненный токен в системном хранилище и передать его в
	// менеджер конфигурации
	if token, err := storage.GetToken(); err == nil && token != "" {
		// Устанавливаем токен в память конфига, чтобы gRPC клиент мог его
		// использовать
		_ = ConfigManager.SetToken(token)
	}

	// 2. Инициализация локальной БД BoltDB
	LocalStorage, err = storage.NewStorage(ConfigManager.GetStoragePath(), ConfigManager.GetLocalCacheSize())
	if err != nil {
		return fmt.Errorf("failed to init local storage: %w", err)
	}

	return nil
}
