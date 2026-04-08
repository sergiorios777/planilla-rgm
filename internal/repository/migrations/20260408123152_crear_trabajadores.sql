-- +goose Up
-- +goose StatementBegin
CREATE TABLE trabajadores (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    tipo_documento VARCHAR(20) NOT NULL DEFAULT 'DNI', -- DNI, CE, Pasaporte
    numero_documento VARCHAR(20) NOT NULL,
    
    nombres VARCHAR(100) NOT NULL,
    apellido_paterno VARCHAR(100) NOT NULL,
    apellido_materno VARCHAR(100) NOT NULL,
    
    fecha_nacimiento DATE,
    sexo VARCHAR(1), -- 'M' o 'F'
    
    activo BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Restricción vital: Un mismo DNI no puede estar registrado dos veces en la MISMA municipalidad
    CONSTRAINT unique_documento_tenant UNIQUE(tenant_id, tipo_documento, numero_documento)
);

-- Creamos un índice para acelerar las búsquedas por municipalidad, ya que siempre filtraremos por ahí
CREATE INDEX idx_trabajadores_tenant ON trabajadores(tenant_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS trabajadores;
-- +goose StatementEnd
