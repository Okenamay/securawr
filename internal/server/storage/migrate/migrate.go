package migrate

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// RunMigrations выполняет миграции базы данных при помощи Goose
func RunMigrations(db *sql.DB, log *zap.Logger) error {
	// Указываем диалект PostgreSQL
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect error: %w", err)
	}

	// Устанавливаем встроенную файловую систему
	// Goose теперь будет искать файлы не на диске, а внутри переменной embedMigrations
	goose.SetBaseFS(embedMigrations)

	// Путь к папке с миграциями внутри embed.FS
	// Так как мы указали //go:embed migrations/*.sql, файлы лежат в виртуальной папке "migrations"
	migrationsDir := "migrations"

	log.Info("Running migrations", zap.String("dir", migrationsDir))

	// Запуск миграций
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("goose up error: %w", err)
	}

	log.Info("Migrations applied successfully")
	return nil
}
