-- +goose Up
-- +goose StatementBegin

-- 1. FASE 1.1: Tabla de Metas Presupuestales
CREATE TABLE metas_presupuestales (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    anio INTEGER NOT NULL,
    codigo VARCHAR(20) NOT NULL,
    descripcion VARCHAR(255) NOT NULL,
    activo BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Una municipalidad no puede repetir el código de meta en el mismo año
    CONSTRAINT unique_meta_anio_tenant UNIQUE(tenant_id, anio, codigo)
);

CREATE INDEX idx_metas_tenant_anio ON metas_presupuestales(tenant_id, anio);

-- 2. FASE 1.2: Tabla de Puestos (Plazas del CAP/CPE)
CREATE TABLE puestos (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- El puesto se amarra al presupuesto y al régimen normativo
    meta_id INTEGER NOT NULL REFERENCES metas_presupuestales(id),
    fuente_rubro_id INTEGER NOT NULL REFERENCES fuentes_rubros(id),
    regimen_id INTEGER NOT NULL REFERENCES regimenes_laborales(id),
    
    nombre VARCHAR(150) NOT NULL, -- Ej. "Especialista en Logística II"
    sueldo_presupuestado NUMERIC(10, 2) NOT NULL, -- Cuánto nos cuesta esta plaza al mes
    estado VARCHAR(20) DEFAULT 'VACANTE', -- 'VACANTE' u 'OCUPADO'
    
    activo BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_puestos_tenant ON puestos(tenant_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS puestos;
DROP TABLE IF EXISTS metas_presupuestales;
-- +goose StatementEnd
