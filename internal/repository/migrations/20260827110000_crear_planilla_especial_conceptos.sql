-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS planilla_especial_conceptos (
    id SERIAL PRIMARY KEY,
    planilla_id INTEGER NOT NULL REFERENCES planillas(id) ON DELETE CASCADE,
    concepto_tenant_id INTEGER NOT NULL REFERENCES conceptos_tenant(id) ON DELETE CASCADE,
    monto_base NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_planilla_especial_concepto UNIQUE(planilla_id, concepto_tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_planilla_especial_conceptos_planilla ON planilla_especial_conceptos(planilla_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS planilla_especial_conceptos;
-- +goose StatementEnd
