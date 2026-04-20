-- +goose Up
-- Agregar columna EsExtraordinario como boolean DEFAULT false
ALTER TABLE conceptos_tenant ADD COLUMN es_extraordinario BOOLEAN DEFAULT false;
-- Agregar columna created_at y updated_at
ALTER TABLE conceptos_tenant ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE conceptos_tenant ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- +goose Down
-- Eliminar columna EsExtraordinario
ALTER TABLE conceptos_tenant DROP COLUMN es_extraordinario;
-- Eliminar columna created_at y updated_at
ALTER TABLE conceptos_tenant DROP COLUMN created_at;
ALTER TABLE conceptos_tenant DROP COLUMN updated_at;
