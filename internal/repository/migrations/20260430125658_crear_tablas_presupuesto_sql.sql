-- +goose Up
-- +goose StatementBegin

-- 1. Añadimos la columna a la tabla puestos
ALTER TABLE puestos ADD COLUMN es_dietario BOOLEAN DEFAULT FALSE;

-- 2. Añadimos el concepto maestro para las Dietas
INSERT INTO conceptos_maestros (codigo, descripcion, tipo, activo)
VALUES ('S102', 'Dietas', 'INGRESO', true);

-- 3. Creamos la tabla para guardar las versiones del Presupuesto
CREATE TABLE pap_versiones (
    id SERIAL PRIMARY KEY,
    tenant_id INT NOT NULL,
    anio INT NOT NULL,
    tipo VARCHAR(50) NOT NULL,
    fecha_generacion TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    estado VARCHAR(20) DEFAULT 'CERRADA'
);

-- 4. Creamos la tabla para el detalle matricial con sus descripciones
CREATE TABLE pap_detalles (
    id SERIAL PRIMARY KEY,
    version_id INT REFERENCES pap_versiones(id) ON DELETE CASCADE,
    
    meta_codigo VARCHAR(50),
    meta_descripcion VARCHAR(255),
    
    fuente_rubro_codigo VARCHAR(50),
    fuente_rubro_descripcion VARCHAR(255),
    
    clasificador_codigo_limpio VARCHAR(50),
    clasificador_descripcion VARCHAR(255),
    
    mes_01 NUMERIC(10,2) DEFAULT 0,
    mes_02 NUMERIC(10,2) DEFAULT 0,
    mes_03 NUMERIC(10,2) DEFAULT 0,
    mes_04 NUMERIC(10,2) DEFAULT 0,
    mes_05 NUMERIC(10,2) DEFAULT 0,
    mes_06 NUMERIC(10,2) DEFAULT 0,
    mes_07 NUMERIC(10,2) DEFAULT 0,
    mes_08 NUMERIC(10,2) DEFAULT 0,
    mes_09 NUMERIC(10,2) DEFAULT 0,
    mes_10 NUMERIC(10,2) DEFAULT 0,
    mes_11 NUMERIC(10,2) DEFAULT 0,
    mes_12 NUMERIC(10,2) DEFAULT 0,
    total_anual NUMERIC(12,2) DEFAULT 0
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Aquí colocamos las instrucciones inversas por si necesitamos deshacer los cambios
DROP TABLE IF EXISTS pap_detalles;
DROP TABLE IF EXISTS pap_versiones;
DELETE FROM conceptos_maestros WHERE codigo = 'S102';
ALTER TABLE puestos DROP COLUMN es_dietario;

-- +goose StatementEnd