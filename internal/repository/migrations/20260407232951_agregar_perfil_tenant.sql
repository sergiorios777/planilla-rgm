-- +goose Up
-- +goose StatementBegin
ALTER TABLE tenants 
ADD COLUMN direccion VARCHAR(255),
ADD COLUMN frase_gestion VARCHAR(255),
ADD COLUMN logo_url VARCHAR(255),
ADD COLUMN slug VARCHAR(100) UNIQUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenants 
DROP COLUMN slug,
DROP COLUMN logo_url,
DROP COLUMN frase_gestion,
DROP COLUMN direccion;
-- +goose StatementEnd
