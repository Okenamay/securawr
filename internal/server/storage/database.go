package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"go.uber.org/zap"

	"github.com/Okenamay/securawr/internal/server/config"
	"github.com/Okenamay/securawr/internal/server/storage/migrate"
)

// Storage - структура для работы с базой данных через pgxpool
type Storage struct {
	Pool *pgxpool.Pool
	log  *zap.Logger
}

// New инициализирует пул соединений с БД и запускает миграции
func New(conf *config.Config, log *zap.Logger) (*Storage, error) {
	// Создаем контекст с таймаутом для подключения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Парсим конфигурацию пула (позволяет настроить макс. кол-во соединений и т.д. через DSN)
	config, err := pgxpool.ParseConfig(conf.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pgx config: %w", err)
	}

	// Инициализируем пул
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	// Проверяем соединение (Ping)
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	log.Info("Connected to database via pgxpool")

	stdDB := stdlib.OpenDBFromPool(pool)

	if err := migrate.RunMigrations(stdDB, log); err != nil {
		// Если миграция упала, закрываем пул
		pool.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return &Storage{
		Pool: pool,
		log:  log,
	}, nil
}

// Close закрывает пул соединений
func (s *Storage) Close() {
	if s.Pool != nil {
		s.Pool.Close()
		s.log.Info("DB connection pool closed")
	}
}

// Пример метода для пинга БД (Ping)
func (s *Storage) Ping(ctx context.Context) error {
	return s.Pool.Ping(ctx)
}
