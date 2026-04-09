-- +goose Up
-- +goose StatementBegin

-- ====================================================================
-- 1. Conceptos personalizados por Municipalidad (Tenant)
-- ====================================================================
CREATE TABLE conceptos_tenant (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- AQUÍ ESTÁ LA CORRECCIÓN: Apunta a tu tabla conceptos_maestros
    concepto_id INTEGER NOT NULL REFERENCES conceptos_maestros(id), 
    
    nombre_personalizado VARCHAR(150) NOT NULL, 
    frecuencia_meses VARCHAR(50) DEFAULT '1,2,3,4,5,6,7,8,9,10,11,12', 
    clasificador VARCHAR(50), 
    
    activo BOOLEAN DEFAULT TRUE,
    
    -- Una municipalidad no puede configurar el mismo concepto base dos veces
    CONSTRAINT unique_concepto_tenant UNIQUE(tenant_id, concepto_id)
);

CREATE INDEX idx_conceptos_tenant_tid ON conceptos_tenant(tenant_id);

-- ====================================================================
-- 2. Asignación de Conceptos a la Plaza (Puesto)
-- ====================================================================
CREATE TABLE puesto_conceptos (
    id SERIAL PRIMARY KEY,
    puesto_id INTEGER NOT NULL REFERENCES puestos(id) ON DELETE CASCADE,
    concepto_tenant_id INTEGER NOT NULL REFERENCES conceptos_tenant(id) ON DELETE CASCADE,
    monto NUMERIC(10, 2), 
    activo BOOLEAN DEFAULT TRUE,
    
    -- Un puesto no puede tener el mismo concepto duplicado
    CONSTRAINT unique_puesto_concepto UNIQUE(puesto_id, concepto_tenant_id)
);

CREATE INDEX idx_puesto_conceptos_pid ON puesto_conceptos(puesto_id);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS puesto_conceptos;
DROP TABLE IF EXISTS conceptos_tenant;
-- +goose StatementEnd
