-- +goose Up
-- +goose StatementBegin
-- 1. Eliminar la restricción UNIQUE del código original SUNAT para permitir agrupamiento
ALTER TABLE conceptos_maestros DROP CONSTRAINT IF EXISTS conceptos_maestros_codigo_key;

-- 2. Crear la columna para el código interno (espejo)
ALTER TABLE conceptos_maestros ADD COLUMN codigo_interno VARCHAR(50);

-- 3. Inicializar la nueva columna con el código actual para todos los conceptos existentes
UPDATE conceptos_maestros SET codigo_interno = codigo;

-- 4. Establecer la columna como obligatoria (NOT NULL)
ALTER TABLE conceptos_maestros ALTER COLUMN codigo_interno SET NOT NULL;

-- 5. Crear la restricción de unicidad para la nueva columna del motor de cálculos
ALTER TABLE conceptos_maestros ADD CONSTRAINT conceptos_maestros_codigo_interno_key UNIQUE (codigo_interno);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE conceptos_maestros DROP CONSTRAINT IF EXISTS conceptos_maestros_codigo_interno_key;
ALTER TABLE conceptos_maestros DROP COLUMN IF EXISTS codigo_interno;
ALTER TABLE conceptos_maestros ADD CONSTRAINT conceptos_maestros_codigo_key UNIQUE (codigo);
-- +goose StatementEnd
