-- +goose Up
-- +goose StatementBegin

-- 1. Limpiamos datos de prueba que ya no son compatibles
TRUNCATE TABLE contratos CASCADE;

-- 2. Eliminamos las columnas que ahora le pertenecen al Puesto
ALTER TABLE contratos 
    DROP COLUMN cargo,
    DROP COLUMN sueldo_base,
    DROP COLUMN regimen_id;

-- 3. Agregamos la columna que vincula el contrato con la Plaza
ALTER TABLE contratos 
    ADD COLUMN puesto_id INTEGER NOT NULL REFERENCES puestos(id);

CREATE INDEX idx_contratos_puesto ON contratos(puesto_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE contratos 
    DROP COLUMN puesto_id,
    ADD COLUMN cargo VARCHAR(150),
    ADD COLUMN sueldo_base NUMERIC(10, 2),
    ADD COLUMN regimen_id INTEGER;
-- +goose StatementEnd
