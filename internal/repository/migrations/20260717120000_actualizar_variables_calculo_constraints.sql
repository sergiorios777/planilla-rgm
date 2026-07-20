-- +goose Up
-- +goose StatementBegin

-- 1. Actualizar restricción en la tabla base_regimen_default
ALTER TABLE base_regimen_default DROP CONSTRAINT IF EXISTS chk_variable_calculo_default;
ALTER TABLE base_regimen_default ADD CONSTRAINT chk_variable_calculo_default CHECK (variable_calculo IN (
    'SUELDO_BASICO', 
    'ASIGNACION_FAMILIAR', 
    'SEXTO_GRATIFICACION', 
    'REMUNERACION_VARIABLE',
    'REMUNERACION_COMPUTABLE',
    'MUC',
    'BET',
    'RETRIBUCION_MENSUAL',
    'VALORIZACION_PRINCIPAL',
    'VALORIZACION_AJUSTADA'
));

-- 2. Actualizar restricción en la tabla base_regimen_tenant
ALTER TABLE base_regimen_tenant DROP CONSTRAINT IF EXISTS chk_variable_calculo_tenant;
ALTER TABLE base_regimen_tenant ADD CONSTRAINT chk_variable_calculo_tenant CHECK (variable_calculo IN (
    'SUELDO_BASICO', 
    'ASIGNACION_FAMILIAR', 
    'SEXTO_GRATIFICACION', 
    'REMUNERACION_VARIABLE',
    'REMUNERACION_COMPUTABLE',
    'MUC',
    'BET',
    'RETRIBUCION_MENSUAL',
    'VALORIZACION_PRINCIPAL',
    'VALORIZACION_AJUSTADA'
));

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Revertir las restricciones a su estado original
ALTER TABLE base_regimen_default DROP CONSTRAINT IF EXISTS chk_variable_calculo_default;
ALTER TABLE base_regimen_default ADD CONSTRAINT chk_variable_calculo_default CHECK (variable_calculo IN (
    'SUELDO_BASICO', 
    'ASIGNACION_FAMILIAR', 
    'SEXTO_GRATIFICACION', 
    'REMUNERACION_VARIABLE',
    'REMUNERACION_COMPUTABLE'
));

ALTER TABLE base_regimen_tenant DROP CONSTRAINT IF EXISTS chk_variable_calculo_tenant;
ALTER TABLE base_regimen_tenant ADD CONSTRAINT chk_variable_calculo_tenant CHECK (variable_calculo IN (
    'SUELDO_BASICO', 
    'ASIGNACION_FAMILIAR', 
    'SEXTO_GRATIFICACION', 
    'REMUNERACION_VARIABLE',
    'REMUNERACION_COMPUTABLE'
));

-- +goose StatementEnd
