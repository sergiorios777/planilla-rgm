-- +goose Up
-- +goose StatementBegin
ALTER TABLE fuentes_rubros ADD COLUMN codigo_fuente_rubro VARCHAR(20);
UPDATE fuentes_rubros 
SET codigo_fuente_rubro = SUBSTRING(fuente_financiamiento FROM 1 FOR 2) || SUBSTRING(rubro FROM 1 FOR 2);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE fuentes_rubros DROP COLUMN IF EXISTS codigo_fuente_rubro;
-- +goose StatementEnd
