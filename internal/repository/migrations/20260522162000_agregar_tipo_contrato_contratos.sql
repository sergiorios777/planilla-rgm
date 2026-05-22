-- +goose Up
-- +goose StatementBegin
ALTER TABLE contratos ADD COLUMN tipo_contrato VARCHAR(100);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE contratos DROP COLUMN IF EXISTS tipo_contrato;
-- +goose StatementEnd
