-- +goose Up
-- +goose StatementBegin

-- 1. Agregar columna tipo a la tabla planillas
ALTER TABLE public.planillas 
    ADD COLUMN IF NOT EXISTS tipo VARCHAR(30) DEFAULT 'ORDINARIA' NOT NULL;

-- 2. Migrar datos existentes basados en es_extraordinaria
UPDATE public.planillas 
SET tipo = 'EXTRAORDINARIA' 
WHERE es_extraordinaria = true;

UPDATE public.planillas 
SET tipo = 'ORDINARIA' 
WHERE es_extraordinaria = false OR es_extraordinaria IS NULL;

-- 3. Agregar restricción CHECK si no existe
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_planillas_tipo'
    ) THEN
        ALTER TABLE public.planillas 
            ADD CONSTRAINT chk_planillas_tipo CHECK (tipo IN ('ORDINARIA', 'EXTRAORDINARIA', 'CTS', 'CESE'));
    END IF;
END $$;

-- 4. Agregar columna planilla_id a planillas_cts con clave foránea en cascada
ALTER TABLE public.planillas_cts 
    ADD COLUMN IF NOT EXISTS planilla_id INT REFERENCES public.planillas(id) ON DELETE CASCADE;

-- 5. Crear índices de búsqueda y rendimiento
CREATE INDEX IF NOT EXISTS idx_planillas_tenant_tipo ON public.planillas(tenant_id, tipo, anio, mes);
CREATE INDEX IF NOT EXISTS idx_planillas_cts_planilla_id ON public.planillas_cts(planilla_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS public.idx_planillas_cts_planilla_id;
DROP INDEX IF EXISTS public.idx_planillas_tenant_tipo;
ALTER TABLE public.planillas_cts DROP COLUMN IF EXISTS planilla_id;
ALTER TABLE public.planillas DROP CONSTRAINT IF EXISTS chk_planillas_tipo;
ALTER TABLE public.planillas DROP COLUMN IF EXISTS tipo;
-- +goose StatementEnd
