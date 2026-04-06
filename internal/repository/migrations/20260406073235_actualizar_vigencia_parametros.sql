-- +goose Up
-- +goose StatementBegin

-- 1. Limpiamos la tabla por si hiciste pruebas, para evitar conflictos al alterar columnas
TRUNCATE TABLE parametros_globales;

-- 2. Eliminamos la restricción anterior y la columna año
ALTER TABLE parametros_globales DROP CONSTRAINT unique_parametro_anio;
ALTER TABLE parametros_globales DROP COLUMN anio;

-- 3. Agregamos las nuevas columnas de fecha (tipo DATE)
ALTER TABLE parametros_globales ADD COLUMN fecha_desde DATE NOT NULL;
ALTER TABLE parametros_globales ADD COLUMN fecha_hasta DATE; -- Permite NULL

-- 4. Nueva restricción: No podemos tener la misma clave iniciando exactamente el mismo día
ALTER TABLE parametros_globales ADD CONSTRAINT unique_clave_fecha UNIQUE(clave, fecha_desde);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Reversión
ALTER TABLE parametros_globales DROP CONSTRAINT unique_clave_fecha;
ALTER TABLE parametros_globales DROP COLUMN fecha_hasta;
ALTER TABLE parametros_globales DROP COLUMN fecha_desde;
ALTER TABLE parametros_globales ADD COLUMN anio INTEGER NOT NULL DEFAULT 2026;
ALTER TABLE parametros_globales ADD CONSTRAINT unique_parametro_anio UNIQUE(anio, clave);
-- +goose StatementEnd
