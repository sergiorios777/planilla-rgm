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

-- 6. Agregar la columna updated_at
ALTER TABLE conceptos_modelo ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

-- 7. Agregar la columna es_pensionable
ALTER TABLE conceptos_modelo ADD COLUMN es_pensionable BOOLEAN DEFAULT false;

-- 8. Agregar la columna es_remunerativa
ALTER TABLE conceptos_modelo ADD COLUMN es_remunerativa BOOLEAN DEFAULT false;

-- 9. Agregar la columna es_base_cts
ALTER TABLE conceptos_modelo ADD COLUMN es_base_cts BOOLEAN DEFAULT false;

-- 10. Agregar la columna es_base_beneficios_sociales
ALTER TABLE conceptos_modelo ADD COLUMN es_base_beneficios_sociales BOOLEAN DEFAULT false;


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE regimen_concepto_modelo;
ALTER TABLE conceptos_modelo DROP CONSTRAINT IF EXISTS unique_nombre_modelo;
ALTER TABLE conceptos_modelo ADD COLUMN regimen_id INTEGER REFERENCES regimenes_laborales(id) ON DELETE CASCADE;
ALTER TABLE conceptos_modelo RENAME COLUMN nombre_personalizado TO descripcion;
ALTER TABLE conceptos_tenant DROP COLUMN modelo_id;
ALTER TABLE conceptos_modelo DROP COLUMN updated_at;
ALTER TABLE conceptos_modelo DROP COLUMN es_pensionable;
ALTER TABLE conceptos_modelo DROP COLUMN es_remunerativa;
ALTER TABLE conceptos_modelo DROP COLUMN es_base_cts;
ALTER TABLE conceptos_modelo DROP COLUMN es_base_beneficios_sociales;

-- +goose StatementEnd
