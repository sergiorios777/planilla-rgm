-- Migración: Módulo de Descuentos y Retenciones Judiciales
-- Archivo: 20260825000000_modulo_descuentos_retenciones.sql

-- 1. Tabla de Entidades Financieras (Tabla 3 SUNAT)
CREATE TABLE IF NOT EXISTS public.entidades_financieras (
    id SERIAL PRIMARY KEY,
    codigo VARCHAR(10) NOT NULL UNIQUE,
    nombre VARCHAR(150) NOT NULL,
    activo BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Precarga de Entidades Financieras SUNAT
INSERT INTO public.entidades_financieras (codigo, nombre, activo) VALUES
('002', 'BANCO DE CRÉDITO DEL PERÚ (BCP)', true),
('003', 'INTERBANK', true),
('007', 'CITIBANK DEL PERÚ', true),
('009', 'SCOTIABANK PERÚ', true),
('011', 'BBVA BANCO CONTINENTAL', true),
('018', 'BANCO DE LA NACIÓN', true),
('020', 'BANCO FALABELLA', true),
('023', 'BANCO DE COMERCIO', true),
('035', 'BANCO PICHINCHA', true),
('038', 'BANCO INTERAMERICANO DE FINANZAS (BANBIF)', true),
('043', 'CREDISCOTIA FINANCIERA', true),
('053', 'BANCO GNB', true),
('056', 'BANCO SANTANDER PERÚ', true),
('057', 'BANCO AZTECA', true),
('058', 'BANCO CENCOSUD', true),
('059', 'BANCO RIPLEY', true),
('060', 'ICBC PERÚ BANK', true),
('070', 'MIBANCO', true),
('200', 'FINANCIERA CREDINKA', true),
('202', 'FINANCIERA PROEMPRESA', true),
('204', 'FINANCIERA CONFIANZA', true),
('206', 'FINANCIERA CREDIRAIZ', true),
('208', 'COMPARTAMOS FINANCIERA', true),
('210', 'FINANCIERA QAPAQ', true),
('212', 'FINANCIERA TFC', true),
('214', 'FINANCIERA EFECTIVA', true),
('216', 'AMERIKA FINANCIERA', true),
('218', 'FINANCIERA OH!', true),
('800', 'CAJA METROPOLITANA DE LIMA', true),
('802', 'CMAC TRUJILLO', true),
('803', 'CMAC AREQUIPA', true),
('805', 'CMAC SULLANA', true),
('806', 'CMAC CUSCO', true),
('808', 'CMAC HUANCAYO', true),
('813', 'CMAC TACNA', true),
('820', 'CMAC DEL SANTA', true),
('822', 'CMAC ICA', true),
('824', 'CMAC PIURA', true),
('826', 'CMAC MAYNAS', true),
('828', 'CMAC PAITA', true),
('900', 'CRAC SIPAN', true),
('902', 'CRAC DEL CENTRO', true),
('904', 'CRAC INCASUR', true),
('906', 'CRAC PRYMERA', true),
('908', 'CRAC LOS ANDES', true)
ON CONFLICT (codigo) DO UPDATE SET nombre = EXCLUDED.nombre, activo = EXCLUDED.activo;

-- 2. Tabla Maestra de Descuentos y Retenciones por Trabajador
CREATE TABLE IF NOT EXISTS public.descuentos (
    id SERIAL PRIMARY KEY,
    tenant_id INT NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    trabajador_id INT NOT NULL REFERENCES public.trabajadores(id) ON DELETE CASCADE,
    concepto_tenant_id INT NOT NULL REFERENCES public.conceptos_tenant(id) ON DELETE RESTRICT,
    
    tipo_descuento VARCHAR(50) NOT NULL DEFAULT 'JUDICIAL',
    documento_ordenador VARCHAR(50) NOT NULL DEFAULT 'RESOLUCION',
    detalle_documento VARCHAR(255),
    descripcion VARCHAR(255) NOT NULL,
    
    tipo_calculo VARCHAR(20) NOT NULL DEFAULT 'PORCENTAJE',
    base_calculo VARCHAR(20) NOT NULL DEFAULT 'NETO_LEY',
    porcentaje NUMERIC(5,2) DEFAULT 0.00,
    monto_fijo NUMERIC(10,2) DEFAULT 0.00,
    
    monto_total_deuda NUMERIC(10,2) DEFAULT 0.00,
    monto_acumulado NUMERIC(10,2) DEFAULT 0.00,
    cuotas_totales INT DEFAULT 0,
    cuota_actual INT DEFAULT 0,

    inicio_vigencia DATE NOT NULL,
    fin_vigencia DATE,
    activo BOOLEAN DEFAULT true NOT NULL,
    motivo_baja VARCHAR(255),

    beneficiario_tipo_documento VARCHAR(20) DEFAULT 'DNI',
    beneficiario_numero_documento VARCHAR(20),
    beneficiario_nombre VARCHAR(200),
    entidad_financiera_id INT REFERENCES public.entidades_financieras(id) ON DELETE SET NULL,
    beneficiario_cuenta VARCHAR(50),
    beneficiario_cci VARCHAR(50),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_tipo_calculo CHECK (tipo_calculo IN ('PORCENTAJE', 'MONTO_FIJO')),
    CONSTRAINT chk_base_calculo CHECK (base_calculo IN ('NETO_LEY', 'BRUTO_AFECTO')),
    CONSTRAINT chk_tipo_descuento CHECK (tipo_descuento IN ('JUDICIAL', 'SINDICAL', 'PRESTAMO', 'CONVENIO', 'OTROS')),
    CONSTRAINT chk_descuento_monto_porcentaje CHECK (
        (tipo_calculo = 'PORCENTAJE' AND porcentaje > 0) OR 
        (tipo_calculo = 'MONTO_FIJO' AND monto_fijo > 0)
    )
);

-- 3. Base Imponible: Conceptos de Ingreso Afectos al Descuento
CREATE TABLE IF NOT EXISTS public.descuento_conceptos (
    id SERIAL PRIMARY KEY,
    descuento_id INT NOT NULL REFERENCES public.descuentos(id) ON DELETE CASCADE,
    concepto_tenant_id INT NOT NULL REFERENCES public.conceptos_tenant(id) ON DELETE RESTRICT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_descuento_concepto UNIQUE (descuento_id, concepto_tenant_id)
);

-- Índices de consulta rápida
CREATE INDEX IF NOT EXISTS idx_descuentos_tenant_trabajador ON public.descuentos(tenant_id, trabajador_id, activo);
CREATE INDEX IF NOT EXISTS idx_descuentos_vigencia ON public.descuentos(tenant_id, inicio_vigencia, fin_vigencia);
CREATE INDEX IF NOT EXISTS idx_descuento_conceptos_descuento ON public.descuento_conceptos(descuento_id);
