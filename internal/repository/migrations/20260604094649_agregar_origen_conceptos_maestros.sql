-- +goose Up
-- +goose StatementBegin
-- 1. Agregar la columna origen a conceptos_maestros con valor por defecto 'sunat'
ALTER TABLE conceptos_maestros ADD COLUMN origen VARCHAR(20) DEFAULT 'sunat' NOT NULL;

-- 2. Añadir la restricción CHECK para validar que solo sea 'sunat' o 'interno'
ALTER TABLE conceptos_maestros ADD CONSTRAINT chk_conceptos_maestros_origen CHECK (origen IN ('sunat', 'interno'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE conceptos_maestros DROP CONSTRAINT IF EXISTS chk_conceptos_maestros_origen;
ALTER TABLE conceptos_maestros DROP COLUMN IF EXISTS origen;
-- +goose StatementEnd
