-- +goose Up
-- +goose StatementBegin

-- Creamos la tabla de inquilinos (Municipalidades/Entidades)
CREATE TABLE tenants (
    id SERIAL PRIMARY KEY,
    nombre VARCHAR(255) NOT NULL,
    ruc VARCHAR(11) UNIQUE NOT NULL,
    activo BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Creamos la tabla de usuarios
CREATE TABLE usuarios (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE, -- Clave foránea
    nombre VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    rol VARCHAR(50) NOT NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Para revertir la migración, borramos las tablas en orden inverso
DROP TABLE IF EXISTS usuarios;
DROP TABLE IF EXISTS tenants;

-- +goose StatementEnd
