-- +goose Up
-- +goose StatementBegin
ALTER TABLE puestos ALTER COLUMN meta_id DROP NOT NULL;
ALTER TABLE puestos ALTER COLUMN fuente_rubro_id DROP NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE puestos ALTER COLUMN meta_id SET NOT NULL;
ALTER TABLE puestos ALTER COLUMN fuente_rubro_id SET NOT NULL;
-- +goose StatementEnd
