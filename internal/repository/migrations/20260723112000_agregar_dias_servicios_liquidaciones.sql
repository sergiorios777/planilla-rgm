-- +goose Up
-- +goose StatementBegin
ALTER TABLE liquidaciones_cese ADD COLUMN dias_servicios INT DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE liquidaciones_cese DROP COLUMN IF EXISTS dias_servicios;
-- +goose StatementEnd
