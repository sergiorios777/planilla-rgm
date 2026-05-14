-- +goose Up
-- +goose StatementBegin

-- 1. Eliminamos las restricciones que dependen de regimen_id
ALTER TABLE conceptos_modelo DROP CONSTRAINT IF EXISTS conceptos_modelo_regimen_id_nombre_personalizado_key;
ALTER TABLE conceptos_modelo DROP CONSTRAINT IF EXISTS conceptos_modelo_regimen_id_fkey;

-- 2. Quitamos la columna
ALTER TABLE conceptos_modelo DROP COLUMN regimen_id;

-- 3. Volvemos a crear la restricción única (ahora el nombre debe ser único a nivel global del modelo)
ALTER TABLE conceptos_modelo ADD CONSTRAINT unique_nombre_modelo UNIQUE (nombre_personalizado);

-- 4. Agregamos la columna modelo_id a la tabla conceptos_tenant
ALTER TABLE conceptos_tenant ADD COLUMN modelo_id INTEGER REFERENCES conceptos_modelo(id) ON DELETE CASCADE;

-- 5. Creamos la tabla intermedia para la relación Muchos a Muchos
CREATE TABLE regimen_concepto_modelo (
    regimen_id INTEGER NOT NULL REFERENCES regimenes_laborales(id) ON DELETE CASCADE,
    concepto_modelo_id INTEGER NOT NULL REFERENCES conceptos_modelo(id) ON DELETE CASCADE,
    PRIMARY KEY (regimen_id, concepto_modelo_id)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE regimen_concepto_modelo;
ALTER TABLE conceptos_modelo DROP CONSTRAINT IF EXISTS unique_nombre_modelo;
ALTER TABLE conceptos_modelo ADD COLUMN regimen_id INTEGER REFERENCES regimenes_laborales(id) ON DELETE CASCADE;
ALTER TABLE conceptos_modelo RENAME COLUMN nombre_personalizado TO descripcion;

-- +goose StatementEnd
