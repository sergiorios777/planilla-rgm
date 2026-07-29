-- +goose Up
-- +goose StatementBegin
ALTER TABLE trabajadores ADD COLUMN fecha_cese DATE;
ALTER TABLE trabajadores ADD COLUMN direccion VARCHAR(255);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE trabajadores DROP COLUMN IF EXISTS fecha_cese;
ALTER TABLE trabajadores DROP COLUMN IF EXISTS direccion;
-- +goose StatementEnd
