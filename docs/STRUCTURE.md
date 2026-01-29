# СТРУКТУРА ПРОЕКТА SECURAWR

```
github.com/Okenamay/securawr/
├── cmd/
│   ├── client/
│   │   └── main.go
│   └── server/
│       └── main.go
├── docs/
│   └── STRUCTURE.md                # Этот документ
├── gen/                            # Сгенерированный gRPC код
│   └── proto/
│       ├── securawr.pb.go
│       └── securawr_grpc.pb.go
├── internal/
│   ├── client/
│   │   └── cli/                    # Логика CLI команд (Cobra)
│   │       ├── root.go
│   │       └── version.go
│   ├── grpcserv/
│   │       └── grpcserv.go         # gRPC сервер
│   ├── logger/
│   │   └── zap/
│   │       └── logger.go           # Логер на Zap
│   ├── server/
│   │   ├── auth/
│   │   │   └── argon2.go           # Логика Argon2id и соли
│   │   ├── config/
│   │   │   └── config.go
│   │   ├── handlers/               # gRPC хендлеры
│   │   │   ├── base.go             # Интерфейс сервера
│   │   │   └── models.go           # DTO модели
│   │   └── storage/                # Слой данных
│   │       ├── migrate/
│   │       │   ├── migrations/     # SQL Goose-файлы миграций
│   │       │   │   ├── 20260125100000_init_users.sql
│   │       │   │   └── 20260129145000_create_data_records.sql
│   │       │   └── migrate.go      # Миграция на Goose
│   │       ├── database.go         # Обеспмечение работы с БД
│   │       ├── dbquery.go          # SQL запросы
│   │       └── models.go           # DB модели
│   ├── tlsserv/
│   │       └── tlsserv.go          # TLS-настройки сервера
│   ├── version/
│   │       └── version.go          # Данные о сборке
├── proto/                          # Protobuf определения
│   ├── securawr.proto
│   ├── securawr.pb.go
│   └── securawr_grpc.pb.go
├── go.mod
└── go.sum
```