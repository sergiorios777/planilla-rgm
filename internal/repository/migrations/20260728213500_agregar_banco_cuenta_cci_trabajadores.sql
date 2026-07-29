-- +goose Up
-- +goose StatementBegin
ALTER TABLE trabajadores ADD COLUMN banco VARCHAR(100);
ALTER TABLE trabajadores ADD COLUMN cuenta VARCHAR(50);
ALTER TABLE trabajadores ADD COLUMN cci VARCHAR(50);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE trabajadores DROP COLUMN IF EXISTS banco;
ALTER TABLE trabajadores DROP COLUMN IF EXISTS cuenta;
ALTER TABLE trabajadores DROP COLUMN IF EXISTS cci;
-- +goose StatementEnd
