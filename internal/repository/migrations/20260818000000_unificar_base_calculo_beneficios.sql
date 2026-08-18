-- +goose Up
-- +goose StatementBegin

-- 1. Agregar columna tipo_entidad a la tabla tenants
ALTER TABLE public.tenants 
    ADD COLUMN IF NOT EXISTS tipo_entidad VARCHAR(50) DEFAULT 'GOBIERNO_LOCAL' NOT NULL;

ALTER TABLE public.tenants 
    DROP CONSTRAINT IF EXISTS chk_tenants_tipo_entidad;

ALTER TABLE public.tenants 
    ADD CONSTRAINT chk_tenants_tipo_entidad 
    CHECK (tipo_entidad IN ('GOBIERNO_LOCAL', 'GOBIERNO_REGIONAL', 'GOBIERNO_NACIONAL', 'OTRO'));

-- 1.1 Agregar columna base_calculo_para en conceptos_modelo y conceptos_tenant
ALTER TABLE public.conceptos_modelo 
    ADD COLUMN IF NOT EXISTS base_calculo_para TEXT[] DEFAULT '{}';

ALTER TABLE public.conceptos_tenant 
    ADD COLUMN IF NOT EXISTS base_calculo_para TEXT[] DEFAULT '{}';

-- 2. Actualizar las restricciones CHECK en base_regimen_default y base_regimen_tenant
ALTER TABLE public.base_regimen_default 
    DROP CONSTRAINT IF EXISTS chk_variable_calculo_default;

ALTER TABLE public.base_regimen_default 
    ADD CONSTRAINT chk_variable_calculo_default 
    CHECK (variable_calculo IN (
        'REMUNERACION_BASICA', 
        'ASIGNACION_FAMILIAR', 
        'SEXTO_GRATIFICACION', 
        'REMUNERACION_VARIABLE',
        'REMUNERACION_COMPUTABLE',
        'MUC',
        'BET',
        'BET_FIJO',
        'BET_VARIABLE',
        'RETRIBUCION_MENSUAL',
        'VALORIZACION_PRINCIPAL',
        'VALORIZACION_AJUSTADA'
    ));

ALTER TABLE public.base_regimen_tenant 
    DROP CONSTRAINT IF EXISTS chk_variable_calculo_tenant;

ALTER TABLE public.base_regimen_tenant 
    ADD CONSTRAINT chk_variable_calculo_tenant 
    CHECK (variable_calculo IN (
        'REMUNERACION_BASICA', 
        'ASIGNACION_FAMILIAR', 
        'SEXTO_GRATIFICACION', 
        'REMUNERACION_VARIABLE',
        'REMUNERACION_COMPUTABLE',
        'MUC',
        'BET',
        'BET_FIJO',
        'BET_VARIABLE',
        'RETRIBUCION_MENSUAL',
        'VALORIZACION_PRINCIPAL',
        'VALORIZACION_AJUSTADA'
    ));

-- 3. Insertar conceptos calculados indispensables
INSERT INTO public.conceptos_calculados (nombre, tipo, codigo_interno) VALUES 
('Compensación por Tiempo de Servicios', 'BENEFICIO_SOCIAL', 'CTS'),
('Gratificaciones de Fiestas Patrias y Navidad', 'BENEFICIO_SOCIAL', 'GRATIFICACION'),
('Aguinaldo Sector Público DL 276', 'BENEFICIO_SOCIAL', 'AGUINALDO_276'),
('Asignación por 25 y 30 Años DL 276', 'BENEFICIO_SOCIAL', 'ASIG_25_30'),
('Subsidio por Luto y Sepelio DL 276', 'BENEFICIO_SOCIAL', 'SUBSIDIO_SEPELIO')
ON CONFLICT (codigo_interno) DO NOTHING;

-- 4. Sembrar base_regimen_default para CTS y GRATIFICACION basados en conceptos_modelo existentes
-- A. CTS para D.L. 728 (REMUNERACION_BASICA y ASIGNACION_FAMILIAR)
INSERT INTO public.base_regimen_default (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo)
SELECT 
    (SELECT id FROM public.conceptos_calculados WHERE codigo_interno = 'CTS'),
    rcm.regimen_id,
    cm.id,
    CASE 
        WHEN cm.nombre_personalizado ILIKE '%familiar%' OR cm.nombre_personalizado ILIKE '%asig%fam%' THEN 'ASIGNACION_FAMILIAR'
        ELSE 'REMUNERACION_BASICA'
    END
FROM public.conceptos_modelo cm
INNER JOIN public.regimen_concepto_modelo rcm ON cm.id = rcm.concepto_modelo_id
INNER JOIN public.regimenes_laborales rl ON rcm.regimen_id = rl.id
WHERE rl.codigo = '728' 
  AND (cm.es_base_cts = true OR cm.es_remunerativa = true OR cm.nombre_personalizado ILIKE '%haber%' OR cm.nombre_personalizado ILIKE '%basico%' OR cm.nombre_personalizado ILIKE '%asig%fam%')
ON CONFLICT (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo) DO NOTHING;

-- B. CTS para D.L. 1057 (CAS) (RETRIBUCION_MENSUAL)
INSERT INTO public.base_regimen_default (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo)
SELECT 
    (SELECT id FROM public.conceptos_calculados WHERE codigo_interno = 'CTS'),
    rcm.regimen_id,
    cm.id,
    'RETRIBUCION_MENSUAL'
FROM public.conceptos_modelo cm
INNER JOIN public.regimen_concepto_modelo rcm ON cm.id = rcm.concepto_modelo_id
INNER JOIN public.regimenes_laborales rl ON rcm.regimen_id = rl.id
WHERE rl.codigo IN ('1057', 'CAS')
  AND (cm.es_remunerativa = true OR cm.nombre_personalizado ILIKE '%remunerac%' OR cm.nombre_personalizado ILIKE '%honorario%' OR cm.nombre_personalizado ILIKE '%retribuc%')
ON CONFLICT (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo) DO NOTHING;

-- C. CTS para D.L. 276 (MUC y BET)
INSERT INTO public.base_regimen_default (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo)
SELECT 
    (SELECT id FROM public.conceptos_calculados WHERE codigo_interno = 'CTS'),
    rcm.regimen_id,
    cm.id,
    CASE 
        WHEN cm.nombre_personalizado ILIKE '%bet%' THEN 'BET'
        ELSE 'MUC'
    END
FROM public.conceptos_modelo cm
INNER JOIN public.regimen_concepto_modelo rcm ON cm.id = rcm.concepto_modelo_id
INNER JOIN public.regimenes_laborales rl ON rcm.regimen_id = rl.id
WHERE rl.codigo = '276'
  AND (cm.nombre_personalizado ILIKE '%muc%' OR cm.nombre_personalizado ILIKE '%bet%' OR cm.nombre_personalizado ILIKE '%haber%' OR cm.nombre_personalizado ILIKE '%basico%')
ON CONFLICT (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo) DO NOTHING;

-- D. GRATIFICACION para D.L. 728
INSERT INTO public.base_regimen_default (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo)
SELECT 
    (SELECT id FROM public.conceptos_calculados WHERE codigo_interno = 'GRATIFICACION'),
    rcm.regimen_id,
    cm.id,
    CASE 
        WHEN cm.nombre_personalizado ILIKE '%familiar%' OR cm.nombre_personalizado ILIKE '%asig%fam%' THEN 'ASIGNACION_FAMILIAR'
        ELSE 'REMUNERACION_BASICA'
    END
FROM public.conceptos_modelo cm
INNER JOIN public.regimen_concepto_modelo rcm ON cm.id = rcm.concepto_modelo_id
INNER JOIN public.regimenes_laborales rl ON rcm.regimen_id = rl.id
WHERE rl.codigo = '728' 
  AND (cm.es_base_beneficios_sociales = true OR cm.es_remunerativa = true OR cm.nombre_personalizado ILIKE '%haber%' OR cm.nombre_personalizado ILIKE '%basico%' OR cm.nombre_personalizado ILIKE '%asig%fam%')
ON CONFLICT (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo) DO NOTHING;

-- E. GRATIFICACION para D.L. 1057 (CAS)
INSERT INTO public.base_regimen_default (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo)
SELECT 
    (SELECT id FROM public.conceptos_calculados WHERE codigo_interno = 'GRATIFICACION'),
    rcm.regimen_id,
    cm.id,
    'RETRIBUCION_MENSUAL'
FROM public.conceptos_modelo cm
INNER JOIN public.regimen_concepto_modelo rcm ON cm.id = rcm.concepto_modelo_id
INNER JOIN public.regimenes_laborales rl ON rcm.regimen_id = rl.id
WHERE rl.codigo IN ('1057', 'CAS')
  AND (cm.es_remunerativa = true OR cm.nombre_personalizado ILIKE '%remunerac%' OR cm.nombre_personalizado ILIKE '%honorario%' OR cm.nombre_personalizado ILIKE '%retribuc%')
ON CONFLICT (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo) DO NOTHING;

-- 5. Sembrar automáticamente base_regimen_tenant para todos los tenants activos
INSERT INTO public.base_regimen_tenant (tenant_id, concepto_calculado_id, regimen_id, concepto_tenant_id, variable_calculo, activo)
SELECT 
    ct.tenant_id,
    brd.concepto_calculado_id,
    brd.regimen_id,
    ct.id,
    brd.variable_calculo,
    true
FROM public.base_regimen_default brd
INNER JOIN public.conceptos_tenant ct ON brd.concepto_modelo_id = ct.modelo_id
ON CONFLICT (tenant_id, concepto_calculado_id, regimen_id, concepto_tenant_id, variable_calculo) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM public.base_regimen_tenant WHERE concepto_calculado_id IN (SELECT id FROM public.conceptos_calculados WHERE codigo_interno IN ('CTS', 'GRATIFICACION', 'AGUINALDO_276', 'ASIG_25_30', 'SUBSIDIO_SEPELIO'));
DELETE FROM public.base_regimen_default WHERE concepto_calculado_id IN (SELECT id FROM public.conceptos_calculados WHERE codigo_interno IN ('CTS', 'GRATIFICACION', 'AGUINALDO_276', 'ASIG_25_30', 'SUBSIDIO_SEPELIO'));
DELETE FROM public.conceptos_calculados WHERE codigo_interno IN ('CTS', 'GRATIFICACION', 'AGUINALDO_276', 'ASIG_25_30', 'SUBSIDIO_SEPELIO');
ALTER TABLE public.tenants DROP COLUMN IF EXISTS tipo_entidad;
-- +goose StatementEnd
