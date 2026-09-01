-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS personal_licencias_vacaciones (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    trabajador_id INTEGER NOT NULL REFERENCES trabajadores(id) ON DELETE CASCADE,
    contrato_id INTEGER REFERENCES contratos(id) ON DELETE SET NULL,
    
    tipo VARCHAR(30) NOT NULL, -- 'VACACION', 'LICENCIA_CON_GOCE', 'LICENCIA_SIN_GOCE'
    subtipo VARCHAR(60),        -- 'DESCANSO_MEDICO', 'MATERNIDAD', 'PATERNIDAD', 'LUTO', 'PARTICULAR', etc.
    codigo_sunat_suspension VARCHAR(10) NOT NULL REFERENCES sunat_tipos_suspension(codigo),
    
    fecha_inicio DATE NOT NULL,
    fecha_fin DATE NOT NULL,
    dias_calendario INTEGER GENERATED ALWAYS AS (fecha_fin - fecha_inicio + 1) STORED,
    
    documento_aprobacion VARCHAR(255) NOT NULL,
    fecha_aprobacion DATE,
    observaciones TEXT,
    
    estado VARCHAR(20) NOT NULL DEFAULT 'APROBADO', -- 'PROGRAMADO', 'APROBADO', 'EJECUTADO', 'CANCELADO'
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_fechas_consistentes CHECK (fecha_fin >= fecha_inicio)
);

CREATE INDEX IF NOT EXISTS idx_licencias_vac_tenant_fechas 
ON personal_licencias_vacaciones(tenant_id, trabajador_id, fecha_inicio, fecha_fin);

CREATE INDEX IF NOT EXISTS idx_licencias_vac_periodo 
ON personal_licencias_vacaciones(tenant_id, fecha_inicio, fecha_fin);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS personal_licencias_vacaciones;
-- +goose StatementEnd
