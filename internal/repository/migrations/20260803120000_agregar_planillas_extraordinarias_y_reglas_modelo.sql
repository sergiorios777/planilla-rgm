-- +goose Up
-- +goose StatementBegin
ALTER TABLE planillas
    ADD COLUMN es_extraordinaria BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE reglas_financiamiento_modelo (
    id SERIAL PRIMARY KEY,
    concepto_modelo_id INTEGER NOT NULL REFERENCES conceptos_modelo(id) ON DELETE CASCADE,
    regimen_id INTEGER REFERENCES regimenes_laborales(id),
    meta_id INTEGER REFERENCES metas_presupuestales(id),
    fuente_rubro_id INTEGER REFERENCES fuentes_rubros(id),
    activo BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_reglas_financiamiento_modelo_concepto ON reglas_financiamiento_modelo(concepto_modelo_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_reglas_financiamiento_modelo_concepto;
DROP TABLE IF EXISTS reglas_financiamiento_modelo;
ALTER TABLE planillas
    DROP COLUMN IF EXISTS es_extraordinaria;
-- +goose StatementEnd
