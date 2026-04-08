-- +goose Up
-- +goose StatementBegin

-- 1. Tabla Global de Regímenes Laborales (Leyes Nacionales)
CREATE TABLE regimenes_laborales (
    id SERIAL PRIMARY KEY,
    codigo VARCHAR(10) NOT NULL UNIQUE,
    descripcion VARCHAR(150) NOT NULL
);

-- 2. Tabla de Contratos (Protegida por tenant_id)
CREATE TABLE contratos (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    trabajador_id INTEGER NOT NULL REFERENCES trabajadores(id) ON DELETE CASCADE,
    regimen_id INTEGER NOT NULL REFERENCES regimenes_laborales(id),
    
    cargo VARCHAR(150) NOT NULL,
    sueldo_base NUMERIC(10, 2) NOT NULL,
    fecha_inicio DATE NOT NULL,
    fecha_fin DATE, -- Puede ser NULL para contratos indeterminados (ej. 276 Nombrados)
    
    activo BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Índices para velocidad de búsqueda
CREATE INDEX idx_contratos_tenant ON contratos(tenant_id);
CREATE INDEX idx_contratos_trabajador ON contratos(trabajador_id);

-- 3. Insertar datos maestros de Regímenes Peruanos
INSERT INTO regimenes_laborales (codigo, descripcion) VALUES 
('276', 'D.L. 276 - Carrera Administrativa'),
('728', 'D.L. 728 - Régimen Privado'),
('1057', 'D.L. 1057 - Contrato Administrativo de Servicios (CAS)'),
('30057', 'Ley 30057 - Servicio Civil (SERVIR)');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS contratos;
DROP TABLE IF EXISTS regimenes_laborales;
-- +goose StatementEnd
