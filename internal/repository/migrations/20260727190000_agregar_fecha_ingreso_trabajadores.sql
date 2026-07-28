-- +goose Up
-- +goose StatementBegin
ALTER TABLE trabajadores ADD COLUMN fecha_ingreso DATE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE trabajadores DROP COLUMN IF EXISTS fecha_ingreso;
-- +goose StatementEnd
