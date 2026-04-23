-- +goose Up
-- 1. Tabla Maestra de AFPs (Integra, Prima, Profuturo, Habitat)
CREATE TABLE afps (
    id SERIAL PRIMARY KEY,
    codigo_sbs VARCHAR(10) UNIQUE,
    nombre VARCHAR(50) NOT NULL,
    activo BOOLEAN DEFAULT true
);

-- 2. Vincular al Trabajador (Modificamos tu tabla existente)
-- Nota: Puedes poner esto en 'trabajadores' o en 'contratos'. 
-- Para el sector público, suele ir en el contrato para mantener el historial si el trabajador se va y regresa años después.
ALTER TABLE trabajadores 
ADD COLUMN regimen_pensionario VARCHAR(20) DEFAULT 'ONP', -- 'ONP' o 'AFP'
ADD COLUMN afp_id INTEGER REFERENCES afps(id),
ADD COLUMN afp_tipo_comision VARCHAR(10), -- 'FLUJO' o 'MIXTA'
ADD COLUMN cuspp VARCHAR(20); -- Código Único del SPP (Dato vital para el PDT PLAME)

-- +goose Down
-- 1. Eliminar las columnas de la tabla trabajadores
ALTER TABLE trabajadores 
DROP COLUMN IF EXISTS regimen_pensionario;
DROP COLUMN IF EXISTS afp_id;
DROP COLUMN IF EXISTS afp_tipo_comision;
DROP COLUMN IF EXISTS cuspp;

-- 2. Eliminar la tabla afps
DROP TABLE IF EXISTS afps;
