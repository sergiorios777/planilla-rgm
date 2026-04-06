-- +goose Up
-- +goose StatementBegin
-- Agregamos la columna. ON DELETE SET NULL asegura que si borramos un "padre", 
-- los hijos no se borren, solo se queden "huérfanos" (parent_id = NULL).
ALTER TABLE clasificadores_mef 
ADD COLUMN parent_id INTEGER REFERENCES clasificadores_mef(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE clasificadores_mef DROP COLUMN parent_id;
-- +goose StatementEnd
