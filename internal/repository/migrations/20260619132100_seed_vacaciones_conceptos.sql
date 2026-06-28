-- +goose Up
-- +goose StatementBegin

-- 1. Insertar conceptos calculados
INSERT INTO conceptos_calculados (nombre, tipo, codigo_interno) VALUES 
('Vacaciones Truncas', 'BENEFICIO_SOCIAL', 'VAC_TRUNCAS'),
('Vacaciones No Gozadas', 'BENEFICIO_SOCIAL', 'VAC_NO_GOZADAS')
ON CONFLICT (codigo_interno) DO NOTHING;

-- 2. Insertar plantillas globales (base_regimen_default)
-- Obtenemos los IDs dinámicamente o por hardcodeo controlado. Dado que los IDs de la base de datos de desarrollo
-- coinciden con los del dump, los usaremos directamente, pero con validación de existencia.

-- Mappings para Vacaciones Truncas (VAC_TRUNCAS)
INSERT INTO base_regimen_default (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo)
SELECT 
    (SELECT id FROM conceptos_calculados WHERE codigo_interno = 'VAC_TRUNCAS'),
    r.id,
    cm.id,
    v.var
FROM (VALUES 
    ('276', 107, 'SUELDO_BASICO'),
    ('276', 108, 'SUELDO_BASICO'),
    ('728', 127, 'SUELDO_BASICO'),
    ('728', 128, 'SUELDO_BASICO'),
    ('728', 135, 'ASIGNACION_FAMILIAR'),
    ('728', 136, 'ASIGNACION_FAMILIAR'),
    ('1057', 142, 'SUELDO_BASICO'),
    ('30057', 137, 'SUELDO_BASICO'),
    ('30057', 138, 'SUELDO_BASICO')
) AS v(reg_cod, modelo_id, var)
JOIN regimenes_laborales r ON r.codigo = v.reg_cod
JOIN conceptos_modelo cm ON cm.id = v.modelo_id
ON CONFLICT (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo) DO NOTHING;

-- Mappings para Vacaciones No Gozadas (VAC_NO_GOZADAS)
INSERT INTO base_regimen_default (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo)
SELECT 
    (SELECT id FROM conceptos_calculados WHERE codigo_interno = 'VAC_NO_GOZADAS'),
    r.id,
    cm.id,
    v.var
FROM (VALUES 
    ('276', 107, 'SUELDO_BASICO'),
    ('276', 108, 'SUELDO_BASICO'),
    ('728', 127, 'SUELDO_BASICO'),
    ('728', 128, 'SUELDO_BASICO'),
    ('728', 135, 'ASIGNACION_FAMILIAR'),
    ('728', 136, 'ASIGNACION_FAMILIAR'),
    ('1057', 142, 'SUELDO_BASICO'),
    ('30057', 137, 'SUELDO_BASICO'),
    ('30057', 138, 'SUELDO_BASICO')
) AS v(reg_cod, modelo_id, var)
JOIN regimenes_laborales r ON r.codigo = v.reg_cod
JOIN conceptos_modelo cm ON cm.id = v.modelo_id
ON CONFLICT (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM base_regimen_default WHERE concepto_calculado_id IN (SELECT id FROM conceptos_calculados WHERE codigo_interno IN ('VAC_TRUNCAS', 'VAC_NO_GOZADAS'));
DELETE FROM conceptos_calculados WHERE codigo_interno IN ('VAC_TRUNCAS', 'VAC_NO_GOZADAS');
-- +goose StatementEnd
