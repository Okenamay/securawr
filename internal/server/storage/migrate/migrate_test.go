package migrate

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // Регистрация драйвера pgx для database/sql
	"go.uber.org/zap"
)

// getTestDSN возвращает строку подключения к тестовой БД
func getTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		// Дефолтное значение, совпадающее с другими тестами
		dsn = "postgres://postgres:postgres@localhost:5432/securawr_test?sslmode=disable"
	}
	return dsn
}

func TestRunMigrations(t *testing.T) {
	dsn := getTestDSN()

	// Открываем соединение через стандартный sql.Open, так как RunMigrations принимает *sql.DB
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Проверяем доступность БД
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping DB: %v. Make sure database is running.", err)
	}

	// Очистка БД перед тестом: удаляем таблицы и служебную таблицу goose
	// Это гарантирует, что миграции будут накатываться с нуля (Up)
	_, err = db.Exec("DROP TABLE IF EXISTS data_records, users, goose_db_version CASCADE")
	if err != nil {
		t.Fatalf("Failed to cleanup DB: %v", err)
	}

	logger := zap.NewNop()

	// 1. Запуск миграций
	err = RunMigrations(db, logger)
	if err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// 2. Проверка результата
	// Проверяем, что таблицы физически создались в схеме public
	tables := []string{"users", "data_records"}
	for _, table := range tables {
		var exists bool
		query := "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)"
		err := db.QueryRow(query, table).Scan(&exists)
		if err != nil {
			t.Errorf("Failed to check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("Expected table '%s' to be created, but it does not exist", table)
		}
	}

	// 3. Проверка идемпотентности (повторный запуск не должен ломать ничего)
	err = RunMigrations(db, logger)
	if err != nil {
		t.Errorf("Second run of RunMigrations failed (should be idempotent): %v", err)
	}
}
