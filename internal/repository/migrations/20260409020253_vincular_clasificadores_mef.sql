-- +goose Up
-- +goose StatementBegin
ALTER TABLE conceptos_tenant 
    DROP COLUMN clasificador,
    ADD COLUMN clasificador_id INTEGER REFERENCES clasificadores_mef(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE conceptos_tenant 
    DROP COLUMN clasificador_id,
    ADD COLUMN clasificador VARCHAR(50);
-- +goose StatementEnd
