-- +goose Up
-- +goose StatementBegin

-- 1. Tabla de Organigramas (Versiones de estructura)
CREATE TABLE organigramas (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    documento_aprobacion VARCHAR(200) NOT NULL, -- Ej: "Ordenanza Municipal N° 045-2024"
    descripcion VARCHAR(255),
    fecha_vigencia DATE NOT NULL,
    activo BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_organigramas_tenant ON organigramas(tenant_id);

-- 2. Tabla de Unidades Orgánicas (Árbol Jerárquico)
CREATE TABLE unidades_organicas (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    organigrama_id INTEGER NOT NULL REFERENCES organigramas(id) ON DELETE CASCADE,
    parent_id INTEGER REFERENCES unidades_organicas(id) ON DELETE SET NULL,
    codigo_mef VARCHAR(50),
    nombre VARCHAR(200) NOT NULL,
    tipo VARCHAR(50) NOT NULL, -- Ej: Gerencia, Subgerencia, Oficina
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_organigrama_nombre UNIQUE (organigrama_id, nombre)
);
CREATE INDEX idx_unidades_organicas_org ON unidades_organicas(organigrama_id);

-- 3. Modificaciones a la tabla de Puestos (Vincular al árbol y código AIRHSP)
ALTER TABLE puestos ADD COLUMN unidad_organica_id INTEGER REFERENCES unidades_organicas(id) ON DELETE SET NULL;
ALTER TABLE puestos ADD COLUMN codigo_airhsp VARCHAR(50);

-- 4. Modificaciones a Planilla Detalles (Snapshot de Inmutabilidad)
ALTER TABLE planilla_detalles ADD COLUMN trabajador_nombre_completo VARCHAR(250);
ALTER TABLE planilla_detalles ADD COLUMN trabajador_numero_documento VARCHAR(20);
ALTER TABLE planilla_detalles ADD COLUMN puesto_codigo_airhsp VARCHAR(50);
ALTER TABLE planilla_detalles ADD COLUMN puesto_nombre VARCHAR(200);
ALTER TABLE planilla_detalles ADD COLUMN organigrama_documento_aprobacion VARCHAR(200);
ALTER TABLE planilla_detalles ADD COLUMN unidad_organica_nombre VARCHAR(200);
ALTER TABLE planilla_detalles ADD COLUMN unidad_organica_tipo VARCHAR(50);
ALTER TABLE planilla_detalles ADD COLUMN sueldo_basico_historico NUMERIC(10, 2);

-- 5. Modificaciones a Planilla Conceptos (Snapshot e Inmutabilidad de Conceptos)
ALTER TABLE planilla_conceptos ALTER COLUMN concepto_tenant_id DROP NOT NULL;
-- Reemplazar FK de concepto_tenant_id por una que soporte SET NULL
ALTER TABLE planilla_conceptos DROP CONSTRAINT IF EXISTS planilla_conceptos_concepto_tenant_id_fkey;
ALTER TABLE planilla_conceptos ADD CONSTRAINT planilla_conceptos_concepto_tenant_id_fkey 
    FOREIGN KEY (concepto_tenant_id) REFERENCES conceptos_tenant(id) ON DELETE SET NULL;

ALTER TABLE planilla_conceptos ADD COLUMN codigo_sunat VARCHAR(10);
ALTER TABLE planilla_conceptos ADD COLUMN nombre_en_boleta VARCHAR(150);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE planilla_conceptos DROP COLUMN IF EXISTS codigo_sunat;
ALTER TABLE planilla_conceptos DROP COLUMN IF EXISTS nombre_en_boleta;
ALTER TABLE planilla_conceptos DROP CONSTRAINT IF EXISTS planilla_conceptos_concepto_tenant_id_fkey;
ALTER TABLE planilla_conceptos ADD CONSTRAINT planilla_conceptos_concepto_tenant_id_fkey 
    FOREIGN KEY (concepto_tenant_id) REFERENCES conceptos_tenant(id);
ALTER TABLE planilla_conceptos ALTER COLUMN concepto_tenant_id SET NOT NULL;

ALTER TABLE planilla_detalles DROP COLUMN IF EXISTS trabajador_nombre_completo;
ALTER TABLE planilla_detalles DROP COLUMN IF EXISTS trabajador_numero_documento;
ALTER TABLE planilla_detalles DROP COLUMN IF EXISTS puesto_codigo_airhsp;
ALTER TABLE planilla_detalles DROP COLUMN IF EXISTS puesto_nombre;
ALTER TABLE planilla_detalles DROP COLUMN IF EXISTS organigrama_documento_aprobacion;
ALTER TABLE planilla_detalles DROP COLUMN IF EXISTS unidad_organica_nombre;
ALTER TABLE planilla_detalles DROP COLUMN IF EXISTS unidad_organica_tipo;
ALTER TABLE planilla_detalles DROP COLUMN IF EXISTS sueldo_basico_historico;

ALTER TABLE puestos DROP COLUMN IF EXISTS unidad_organica_id;
ALTER TABLE puestos DROP COLUMN IF EXISTS codigo_airhsp;

DROP TABLE IF EXISTS unidades_organicas;
DROP TABLE IF EXISTS organigramas;
-- +goose StatementEnd
