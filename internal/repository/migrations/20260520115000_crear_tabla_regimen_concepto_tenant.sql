-- +goose Up
-- +goose StatementBegin
CREATE TABLE regimen_concepto_tenant (
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    regimen_id INTEGER NOT NULL REFERENCES regimenes_laborales(id) ON DELETE CASCADE,
    concepto_tenant_id INTEGER NOT NULL REFERENCES conceptos_tenant(id) ON DELETE CASCADE,
    PRIMARY KEY (tenant_id, regimen_id, concepto_tenant_id)
);

CREATE INDEX idx_regimen_concepto_tenant_lookup ON regimen_concepto_tenant(tenant_id, regimen_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS regimen_concepto_tenant;
-- +goose StatementEnd
