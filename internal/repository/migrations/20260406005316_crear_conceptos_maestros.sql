-- +goose Up
-- +goose StatementBegin
CREATE TABLE conceptos_maestros (
    id SERIAL PRIMARY KEY,
    codigo VARCHAR(50) NOT NULL UNIQUE,
    descripcion VARCHAR(255) NOT NULL,
    tipo VARCHAR(50) NOT NULL, -- 'Ingreso', 'Retencion', o 'Aporte'
    activo BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS conceptos_maestros;
-- +goose StatementEnd
