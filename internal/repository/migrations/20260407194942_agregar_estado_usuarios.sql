-- +goose Up
-- +goose StatementBegin
ALTER TABLE usuarios ADD COLUMN activo BOOLEAN DEFAULT TRUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE usuarios DROP COLUMN activo;
-- +goose StatementEnd
