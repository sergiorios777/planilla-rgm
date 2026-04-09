-- +goose Up
-- +goose StatementBegin
-- 1. Eliminamos la restricción que bloqueaba repetir el concepto maestro
ALTER TABLE conceptos_tenant DROP CONSTRAINT IF EXISTS unique_concepto_tenant;

-- 2. Agregamos la nueva restricción: No pueden haber dos nombres iguales en la misma municipalidad
ALTER TABLE conceptos_tenant ADD CONSTRAINT unique_nombre_concepto_tenant UNIQUE(tenant_id, nombre_personalizado);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Revertimos los cambios en caso de error
ALTER TABLE conceptos_tenant DROP CONSTRAINT IF EXISTS unique_nombre_concepto_tenant;
ALTER TABLE conceptos_tenant ADD CONSTRAINT unique_concepto_tenant UNIQUE(tenant_id, concepto_id);
-- +goose StatementEnd
