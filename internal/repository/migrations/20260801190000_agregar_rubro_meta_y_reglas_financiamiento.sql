-- +goose Up
-- +goose StatementBegin
ALTER TABLE planilla_conceptos
    ADD COLUMN fuente_rubro_id INTEGER REFERENCES fuentes_rubros(id),
    ADD COLUMN meta_id INTEGER REFERENCES metas_presupuestales(id);

CREATE INDEX idx_planilla_conceptos_rubro ON planilla_conceptos(fuente_rubro_id);
CREATE INDEX idx_planilla_conceptos_meta ON planilla_conceptos(meta_id);

CREATE TABLE reglas_financiamiento_concepto (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    concepto_tenant_id INTEGER NOT NULL REFERENCES conceptos_tenant(id) ON DELETE CASCADE,
    regimen_id INTEGER REFERENCES regimenes_laborales(id),
    meta_id INTEGER REFERENCES metas_presupuestales(id),
    fuente_rubro_id INTEGER REFERENCES fuentes_rubros(id),
    activo BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_reglas_financiamiento_tenant ON reglas_financiamiento_concepto(tenant_id);
CREATE INDEX idx_reglas_financiamiento_concepto ON reglas_financiamiento_concepto(concepto_tenant_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS reglas_financiamiento_concepto;
DROP INDEX IF EXISTS idx_planilla_conceptos_meta;
DROP INDEX IF EXISTS idx_planilla_conceptos_rubro;
ALTER TABLE planilla_conceptos
    DROP COLUMN IF EXISTS meta_id,
    DROP COLUMN IF EXISTS fuente_rubro_id;
-- +goose StatementEnd
