package migrate

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

// RunMigrations выполняет миграции базы данных при помощи Goose
func RunMigrations(db *sql.DB, log *zap.Logger) error {
	// Указываем диалект PostgreSQL
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect error: %w", err)
	}

	// Путь к папке с миграциями
	migrationsDir := "internal/server/storage/migrations"

	// Проверяем наличие директории перед запуском
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory not found at '%s': %w", migrationsDir, err)
	}

	log.Info("Running migrations", zap.String("dir", migrationsDir))

	// Запуск миграций
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("goose up error: %w", err)
	}

	log.Info("Migrations applied successfully")
	return nil
}
