-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

DROP TABLE IF EXISTS data_records CASCADE;
DROP TABLE IF EXISTS users CASCADE;


CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    auth_salt BYTEA NOT NULL,
    encryption_salt BYTEA NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE data_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    data_type INT NOT NULL,
    data_blob BYTEA,
    meta_info JSONB NOT NULL DEFAULT '{}'::jsonb,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

DROP INDEX IF EXISTS idx_data_records_user_type;
DROP INDEX IF EXISTS idx_data_records_user_id;
CREATE INDEX idx_data_records_user_id ON data_records(user_id);
CREATE INDEX idx_users_login ON users(login);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Восстанавливаем старую схему users для корректного отката
DROP TABLE IF EXISTS users CASCADE;
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    salt TEXT NOT NULL,
    -- encryption_salt
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

DROP TABLE IF EXISTS data_records CASCADE;
CREATE TABLE data_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    data_type INT NOT NULL,
    data_blob BYTEA,
    meta_info JSONB NOT NULL DEFAULT '{}'::jsonb,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Индексы
DROP INDEX IF EXISTS idx_data_records_user_type;
DROP INDEX IF EXISTS idx_data_records_user_id;
CREATE INDEX idx_data_records_user_id ON data_records(user_id);
CREATE INDEX idx_data_records_user_type ON data_records(user_id, data_type);
-- +goose StatementEnd