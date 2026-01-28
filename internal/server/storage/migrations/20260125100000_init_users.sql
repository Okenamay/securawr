-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    login TEXT NOT NULL UNIQUE,
    auth_salt BYTEA NOT NULL,      -- Соль для аутентификации
    encryption_salt BYTEA NOT NULL,-- Соль для генерации MasterKey
    auth_hash TEXT NOT NULL,       -- Хеш пароля (Argon2id + Pepper)
    pepper_version INTEGER DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd