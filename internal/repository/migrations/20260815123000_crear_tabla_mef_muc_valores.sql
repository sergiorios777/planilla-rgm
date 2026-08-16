-- Migration: Crear tabla mef_muc_valores para registrar valores históricos de Monto Único Consolidado
CREATE TABLE IF NOT EXISTS mef_muc_valores (
    id SERIAL PRIMARY KEY,
    norma_legal VARCHAR(150) NOT NULL,
    fecha_norma DATE NOT NULL,
    grupo_ocupacional VARCHAR(50) NOT NULL,
    nivel_remunerativo VARCHAR(20) NOT NULL,
    monto_muc NUMERIC(12,2) NOT NULL DEFAULT 0.00,
    activo BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_mef_muc_norma ON mef_muc_valores(norma_legal);
CREATE INDEX IF NOT EXISTS idx_mef_muc_grupo_nivel ON mef_muc_valores(grupo_ocupacional, nivel_remunerativo);
