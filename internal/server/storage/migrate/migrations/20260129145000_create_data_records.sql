-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Пересоздаем таблицу users под новые требования (UUID, упрощенная соль для Этапа 2)
DROP TABLE IF EXISTS users CASCADE;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL, -- Бывшее auth_hash, теперь просто хеш пароля
    salt TEXT NOT NULL,          -- Auth salt (теперь строка, base64/hex)
    -- encryption_salt пока не добавляем, это задача Этапа 3
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 2. Создаем таблицу для хранения данных
CREATE TABLE IF NOT EXISTS data_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Тип данных (соответствует enum DataType из proto)
    -- 1=Text, 2=Binary, 3=Card, 4=Credentials
    data_type INT NOT NULL,

    -- Сами данные. Храним в BYTEA
    data_blob BYTEA,

    -- Метаданные в JSONB (имя файла, описание, любые кастомные поля)
    meta_info JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- Вспомогательные поля
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_data_records_user_id ON data_records(user_id);
CREATE INDEX IF NOT EXISTS idx_data_records_user_type ON data_records(user_id, data_type);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_data_records_user_type;
DROP INDEX IF EXISTS idx_data_records_user_id;
DROP TABLE IF EXISTS data_records;

-- Восстанавливаем старую схему users для корректного отката
DROP TABLE IF EXISTS users CASCADE;
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    login TEXT NOT NULL UNIQUE,
    auth_salt BYTEA NOT NULL,
    encryption_salt BYTEA NOT NULL,
    auth_hash TEXT NOT NULL,
    pepper_version INTEGER DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
-- +goose StatementEnd
