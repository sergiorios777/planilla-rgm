-- +goose Up
-- +goose StatementBegin
CREATE TABLE parametros_globales (
    id SERIAL PRIMARY KEY,
    anio INTEGER NOT NULL,
    clave VARCHAR(50) NOT NULL, -- Ej: 'UIT', 'RMV', 'ESSALUD_TASA'
    valor NUMERIC(15, 4) NOT NULL, -- NUMERIC es ideal para dinero o porcentajes
    descripcion VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Restricción vital: No puede haber dos valores para la misma clave en el mismo año
    CONSTRAINT unique_parametro_anio UNIQUE(anio, clave)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS parametros_globales;
-- +goose StatementEnd
