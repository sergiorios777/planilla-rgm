-- +goose Up
-- +goose StatementBegin

-- 1. Agregamos el parent_id a los conceptos maestros
ALTER TABLE conceptos_maestros 
ADD COLUMN parent_id INTEGER REFERENCES conceptos_maestros(id) ON DELETE SET NULL;

-- 2. Creamos la tabla intermedia para las afectaciones
CREATE TABLE conceptos_afectaciones (
    id SERIAL PRIMARY KEY,
    concepto_base_id INTEGER NOT NULL REFERENCES conceptos_maestros(id) ON DELETE CASCADE,
    concepto_derivado_id INTEGER NOT NULL REFERENCES conceptos_maestros(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Restricción para evitar duplicar exactamente la misma relación
    CONSTRAINT unique_afectacion UNIQUE(concepto_base_id, concepto_derivado_id)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Para revertir, siempre borramos primero las tablas dependientes
DROP TABLE IF EXISTS conceptos_afectaciones;

-- Luego alteramos la tabla principal para quitar la columna
ALTER TABLE conceptos_maestros DROP COLUMN parent_id;

-- +goose StatementEnd
