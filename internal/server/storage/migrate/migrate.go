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
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect error: %w", err)
	}

	goose.SetBaseFS(embedMigrations)

	migrationsDir := "migrations"

	log.Info("Running migrations", zap.String("dir", migrationsDir))

	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("goose up error: %w", err)
	}

	log.Info("Migrations applied successfully")
	return nil
}
