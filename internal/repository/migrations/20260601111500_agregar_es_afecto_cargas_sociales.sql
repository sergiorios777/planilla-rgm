-- +goose Up
ALTER TABLE conceptos_modelo ADD COLUMN es_afecto_cargas_sociales BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE conceptos_tenant ADD COLUMN es_afecto_cargas_sociales BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE conceptos_tenant DROP COLUMN es_afecto_cargas_sociales;
ALTER TABLE conceptos_modelo DROP COLUMN es_afecto_cargas_sociales;
