-- +goose Up
ALTER TABLE contratos ADD COLUMN nivel VARCHAR(100);

-- +goose Down
ALTER TABLE contratos DROP COLUMN IF EXISTS nivel;
