-- +goose Up
-- +goose StatementBegin

-- 1. NIVEL CABECERA: La Planilla del Mes (Ej. Enero 2026)
CREATE TABLE planillas (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    anio INTEGER NOT NULL,
    mes INTEGER NOT NULL,
    descripcion VARCHAR(255) NOT NULL, -- Ej. "Planilla Principal CAS - Enero"
    estado VARCHAR(20) DEFAULT 'BORRADOR', -- Estados: BORRADOR, APROBADA, CERRADA
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Una entidad no debería duplicar planillas con el mismo nombre en el mismo mes/año
    CONSTRAINT unique_planilla_mes UNIQUE(tenant_id, anio, mes, descripcion)
);

CREATE INDEX idx_planillas_tenant ON planillas(tenant_id, anio, mes);

-- 2. NIVEL DETALLE: El resumen por Trabajador (La Boleta)
CREATE TABLE planilla_detalles (
    id SERIAL PRIMARY KEY,
    planilla_id INTEGER NOT NULL REFERENCES planillas(id) ON DELETE CASCADE,
    
    -- Nos conectamos al contrato, que a su vez nos da al Trabajador y al Puesto(Plaza)
    contrato_id INTEGER NOT NULL REFERENCES contratos(id), 
    
    total_ingresos NUMERIC(10, 2) DEFAULT 0.00,
    total_retenciones NUMERIC(10, 2) DEFAULT 0.00,
    total_aportes NUMERIC(10, 2) DEFAULT 0.00,
    neto_pagar NUMERIC(10, 2) DEFAULT 0.00,
    
    -- Un trabajador (contrato) solo puede tener una boleta en esta planilla exacta
    CONSTRAINT unique_detalle_contrato UNIQUE(planilla_id, contrato_id)
);

-- 3. NIVEL CONCEPTOS: El desglose rubro por rubro de la boleta
CREATE TABLE planilla_conceptos (
    id SERIAL PRIMARY KEY,
    planilla_detalle_id INTEGER NOT NULL REFERENCES planilla_detalles(id) ON DELETE CASCADE,
    concepto_tenant_id INTEGER NOT NULL REFERENCES conceptos_tenant(id),
    
    -- Guardamos el tipo como texto para sumar rápidamente sin tener que hacer JOIN a conceptos maestros
    tipo_concepto VARCHAR(20) NOT NULL, -- 'INGRESO', 'RETENCION', 'APORTE'
    monto NUMERIC(10, 2) NOT NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS planilla_conceptos;
DROP TABLE IF EXISTS planilla_detalles;
DROP TABLE IF EXISTS planillas;
-- +goose StatementEnd
