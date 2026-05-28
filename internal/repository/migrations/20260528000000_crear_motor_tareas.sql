-- +goose Up
-- 1. Tabla de Tareas Programadas (Uso Interno / Automatizaciones)
CREATE TABLE admin_tareas (
    id SERIAL PRIMARY KEY,
    titulo VARCHAR(150) NOT NULL,
    descripcion TEXT,
    recurrencia VARCHAR(20) NOT NULL, -- 'UNICO', 'MENSUAL', 'TRIMESTRAL'
    fecha_vencimiento TIMESTAMP NOT NULL,
    proximo_aviso TIMESTAMP NOT NULL,
    notificado_email BOOLEAN DEFAULT FALSE,
    activo BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Tabla General de Notificaciones (La Cola de Avisos)
-- tenant_id = NULL y usuario_id = NULL representa notificaciones del súper admin
CREATE TABLE notificaciones (
    id SERIAL PRIMARY KEY,
    tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE,
    usuario_id INT REFERENCES usuarios(id) ON DELETE CASCADE,
    titulo VARCHAR(200) NOT NULL,
    mensaje TEXT NOT NULL,
    tipo VARCHAR(50) NOT NULL, -- 'ALERTA_SISTEMA', 'PROCESO_EXITOSO', 'ERROR'
    leido BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Índices críticos para optimizar consultas en tiempo de Polling
CREATE INDEX idx_notificaciones_usuario_leido ON notificaciones(usuario_id, leido);
CREATE INDEX idx_notificaciones_tenant_leido ON notificaciones(tenant_id, leido);
CREATE INDEX idx_admin_tareas_aviso ON admin_tareas(proximo_aviso) WHERE activo = true;

-- +goose Down
DROP TABLE IF EXISTS notificaciones;
DROP TABLE IF EXISTS admin_tareas;
