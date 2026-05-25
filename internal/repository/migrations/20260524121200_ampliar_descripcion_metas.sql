-- Migración: Ampliar límite de descripción de metas presupuestales a 512 caracteres
-- Creado: 2026-05-24

-- +goose Up
-- +goose StatementBegin
ALTER TABLE metas_presupuestales ALTER COLUMN descripcion TYPE character varying(512);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE metas_presupuestales ALTER COLUMN descripcion TYPE character varying(255);
-- +goose StatementEnd
