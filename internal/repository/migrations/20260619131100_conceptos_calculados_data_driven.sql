-- +goose Up
-- +goose StatementBegin

-- 1. Declarar relación lógica en conceptos_tenant para poder usar llave foránea compuesta
ALTER TABLE conceptos_tenant ADD CONSTRAINT unique_tenant_id_id UNIQUE (tenant_id, id);

-- 2. Catálogo de Fórmulas y Beneficios (SaaS Global)
CREATE TABLE conceptos_calculados (
    id SERIAL PRIMARY KEY,
    nombre VARCHAR(150) NOT NULL,
    tipo VARCHAR(50) NOT NULL,             -- Ej: 'BENEFICIO_SOCIAL', 'APORTE', 'IMPUESTO'
    codigo_interno VARCHAR(50) UNIQUE NOT NULL, -- Ej: 'VAC_TRUNCAS', 'VAC_NO_GOZADAS', 'CTS'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 3. Capa Global (La Ley / Súper Admin)
CREATE TABLE base_regimen_default (
    id SERIAL PRIMARY KEY,
    concepto_calculado_id INT NOT NULL REFERENCES conceptos_calculados(id) ON DELETE CASCADE,
    regimen_id INT NOT NULL REFERENCES regimenes_laborales(id) ON DELETE CASCADE,
    concepto_modelo_id INT NOT NULL REFERENCES conceptos_modelo(id) ON DELETE CASCADE,
    variable_calculo VARCHAR(50) NOT NULL, -- Ej: 'SUELDO_BASICO', 'ASIGNACION_FAMILIAR'
    UNIQUE(concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo),
    CONSTRAINT chk_variable_calculo_default CHECK (variable_calculo IN (
        'SUELDO_BASICO', 
        'ASIGNACION_FAMILIAR', 
        'SEXTO_GRATIFICACION', 
        'REMUNERACION_VARIABLE',
        'REMUNERACION_COMPUTABLE'
    ))
);

-- 4. Capa Local (El Cliente / Municipalidad)
CREATE TABLE base_regimen_tenant (
    id SERIAL PRIMARY KEY,
    tenant_id INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    concepto_calculado_id INT NOT NULL REFERENCES conceptos_calculados(id) ON DELETE CASCADE,
    regimen_id INT NOT NULL REFERENCES regimenes_laborales(id) ON DELETE CASCADE,
    concepto_tenant_id INT NOT NULL,
    variable_calculo VARCHAR(50) NOT NULL,
    activo BOOLEAN NOT NULL DEFAULT TRUE, -- Permite excluir conceptos sin que la resiembra los recree
    
    -- Constraint única compuesta
    UNIQUE(tenant_id, concepto_calculado_id, regimen_id, concepto_tenant_id, variable_calculo),
    
    -- Llave foránea compuesta para garantizar aislamiento estricto por tenant
    CONSTRAINT fk_base_regimen_tenant_concepto 
        FOREIGN KEY (tenant_id, concepto_tenant_id) 
        REFERENCES conceptos_tenant(tenant_id, id) ON DELETE CASCADE,
        
    CONSTRAINT chk_variable_calculo_tenant CHECK (variable_calculo IN (
        'SUELDO_BASICO', 
        'ASIGNACION_FAMILIAR', 
        'SEXTO_GRATIFICACION', 
        'REMUNERACION_VARIABLE',
        'REMUNERACION_COMPUTABLE'
    ))
);

-- Índices de Rendimiento para el Motor de Cálculos
CREATE INDEX idx_base_regimen_tenant_calc 
ON base_regimen_tenant(tenant_id, regimen_id, concepto_calculado_id) 
WHERE activo = true;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS base_regimen_tenant;
DROP TABLE IF EXISTS base_regimen_default;
DROP TABLE IF EXISTS conceptos_calculados;
ALTER TABLE conceptos_tenant DROP CONSTRAINT IF EXISTS unique_tenant_id_id;
-- +goose StatementEnd
