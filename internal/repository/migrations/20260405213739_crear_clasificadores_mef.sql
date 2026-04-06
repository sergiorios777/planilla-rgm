-- +goose Up
-- +goose StatementBegin
CREATE TABLE clasificadores_mef (
    id SERIAL PRIMARY KEY,
    anio INTEGER NOT NULL DEFAULT 2026,
    codigo VARCHAR(50) UNIQUE NOT NULL,
    codigo_limpio VARCHAR(50) NOT NULL,
    descripcion VARCHAR(255) NOT NULL,
    nivel INTEGER NOT NULL,
    tipo_transaccion VARCHAR(50) NOT NULL,
    activo BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_anio_codigo UNIQUE (anio, codigo_limpio)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS clasificadores_mef;
-- +goose StatementEnd
