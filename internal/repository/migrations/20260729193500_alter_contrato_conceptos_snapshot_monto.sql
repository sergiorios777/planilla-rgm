-- +goose Up
-- +goose StatementBegin
ALTER TABLE contrato_conceptos_snapshot ALTER COLUMN monto DROP NOT NULL;
ALTER TABLE contrato_conceptos_snapshot ALTER COLUMN monto SET DEFAULT 0.00;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE contrato_conceptos_snapshot ALTER COLUMN monto SET NOT NULL;
ALTER TABLE contrato_conceptos_snapshot ALTER COLUMN monto DROP DEFAULT;
-- +goose StatementEnd
