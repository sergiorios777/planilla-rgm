-- +goose Up
-- +goose StatementBegin

-- 1. Actualizar registros existentes de SUELDO_BASICO a REMUNERACION_BASICA
UPDATE base_regimen_default SET variable_calculo = 'REMUNERACION_BASICA' WHERE variable_calculo = 'SUELDO_BASICO';
UPDATE base_regimen_tenant SET variable_calculo = 'REMUNERACION_BASICA' WHERE variable_calculo = 'SUELDO_BASICO';

-- 2. Cambiar la restricción CHECK en base_regimen_default
ALTER TABLE base_regimen_default DROP CONSTRAINT IF EXISTS chk_variable_calculo_default;
ALTER TABLE base_regimen_default ADD CONSTRAINT chk_variable_calculo_default CHECK (variable_calculo IN (
    'REMUNERACION_BASICA', 
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

-- 3. Cambiar la restricción CHECK en base_regimen_tenant
ALTER TABLE base_regimen_tenant DROP CONSTRAINT IF EXISTS chk_variable_calculo_tenant;
ALTER TABLE base_regimen_tenant ADD CONSTRAINT chk_variable_calculo_tenant CHECK (variable_calculo IN (
    'REMUNERACION_BASICA', 
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

-- 1. Revertir registros existentes de REMUNERACION_BASICA a SUELDO_BASICO
UPDATE base_regimen_default SET variable_calculo = 'SUELDO_BASICO' WHERE variable_calculo = 'REMUNERACION_BASICA';
UPDATE base_regimen_tenant SET variable_calculo = 'SUELDO_BASICO' WHERE variable_calculo = 'REMUNERACION_BASICA';

-- 2. Revertir restricción CHECK en base_regimen_default
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

-- 3. Revertir restricción CHECK en base_regimen_tenant
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
