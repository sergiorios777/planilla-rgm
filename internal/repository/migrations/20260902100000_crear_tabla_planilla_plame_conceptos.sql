-- +goose Up
CREATE TABLE IF NOT EXISTS planilla_plame_conceptos (
    id SERIAL PRIMARY KEY,
    planilla_id INT NOT NULL REFERENCES planillas(id) ON DELETE CASCADE,
    planilla_detalle_id INT NOT NULL REFERENCES planilla_detalles(id) ON DELETE CASCADE,
    trabajador_id INT NOT NULL REFERENCES trabajadores(id) ON DELETE CASCADE,
    planilla_concepto_id INT REFERENCES planilla_conceptos(id) ON DELETE SET NULL,
    
    codigo_sunat VARCHAR(10) NOT NULL,
    descripcion_sunat VARCHAR(255) NOT NULL DEFAULT '',
    tipo_concepto VARCHAR(20) NOT NULL, -- INGRESO, RETENCION, APORTE
    
    monto_devengado NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    monto_pagado NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    
    es_concepto_vacacional BOOLEAN NOT NULL DEFAULT FALSE,
    es_ajuste_manual BOOLEAN NOT NULL DEFAULT FALSE,
    observacion_ajuste TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_plame_conceptos_planilla ON planilla_plame_conceptos(planilla_id);
CREATE INDEX IF NOT EXISTS idx_plame_conceptos_detalle ON planilla_plame_conceptos(planilla_detalle_id);
CREATE INDEX IF NOT EXISTS idx_plame_conceptos_trabajador ON planilla_plame_conceptos(trabajador_id);
CREATE INDEX IF NOT EXISTS idx_plame_conceptos_cod_sunat ON planilla_plame_conceptos(planilla_id, codigo_sunat);

-- +goose Down
DROP TABLE IF EXISTS planilla_plame_conceptos;
