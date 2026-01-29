# СТРУКТУРА ПРОЕКТА SECURAWR

```
github.com/Okenamay/securawr/
├── cmd/
│   ├── client/
│   │   └── main.go                 # Main.go CLI-клиента
│   └── server/
│       └── main.go                 # Main.go сервера
├── gen/                            # Сгенерированный код (Protobuf/gRPC)
│   └── proto/
│       ├── securawr.pb.go
│       └── securawr_grpc.pb.go
├── internal/
│   ├── client/
│   │   └── cli/                    # Логика CLI команд (Cobra)
│   │       ├── root.go             # Базовая команда
│   │       └── version.go          # Команда version
│   ├── logger/
│   │   └── zap/
│   │       └── logger.go           # Логер на Zap
│   └── server/
│       ├── handlers/               # gRPC хендлеры
│       │   ├── base.go             # Интерфейс сервера
│       │   └── models.go           # DTO модели
│       └── storage/                # Слой данных
│           ├── migrate/
│           │   └── migrate.go      # Миграция на Goose
│           ├── migrations/         # SQL Goose-файлы миграций
│           │   └── 20260125100000_init_users.sql
│           ├── database.go         # Обеспмечение работы с БД
│           ├── dbquery.go          # SQL запросы
│           └── models.go           # DB модели
├── proto/                          # Protobuf определения
│   └── securawr.proto
├── go.mod
└── go.sum
```