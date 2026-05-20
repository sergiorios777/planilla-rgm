-- +goose Up
-- +goose StatementBegin
ALTER TABLE conceptos_tenant ADD COLUMN es_pensionable BOOLEAN DEFAULT false;
ALTER TABLE conceptos_tenant ADD COLUMN es_remunerativa BOOLEAN DEFAULT false;
ALTER TABLE conceptos_tenant ADD COLUMN es_base_cts BOOLEAN DEFAULT false;
ALTER TABLE conceptos_tenant ADD COLUMN es_base_beneficios_sociales BOOLEAN DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE conceptos_tenant DROP COLUMN es_pensionable;
ALTER TABLE conceptos_tenant DROP COLUMN es_remunerativa;
ALTER TABLE conceptos_tenant DROP COLUMN es_base_cts;
ALTER TABLE conceptos_tenant DROP COLUMN es_base_beneficios_sociales;
-- +goose StatementEnd
