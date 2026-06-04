-- +goose Up
-- +goose StatementBegin

-- 1. Tabla de cabecera para los cálculos semestrales de CTS (DL 728)
CREATE TABLE planillas_cts (
    id SERIAL PRIMARY KEY,
    tenant_id INT NOT NULL,
    anio INT NOT NULL,
    periodo VARCHAR(20) NOT NULL, -- 'MAYO' (Nov-Abr) o 'NOVIEMBRE' (May-Oct)
    estado VARCHAR(20) DEFAULT 'BORRADOR', -- 'BORRADOR', 'PROCESADA'
    fecha_calculo TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_planilla_cts_periodo UNIQUE(tenant_id, anio, periodo)
);

-- 2. Tabla de detalle de CTS por trabajador (DL 728)
CREATE TABLE planilla_cts_detalles (
    id SERIAL PRIMARY KEY,
    planilla_cts_id INT NOT NULL REFERENCES planillas_cts(id) ON DELETE CASCADE,
    contrato_id INT NOT NULL REFERENCES contratos(id) ON DELETE CASCADE,
    sueldo_basico NUMERIC(10,2) DEFAULT 0,
    asignacion_familiar NUMERIC(10,2) DEFAULT 0,
    sexto_gratificacion NUMERIC(10,2) DEFAULT 0,
    promedio_variables NUMERIC(10,2) DEFAULT 0,
    remuneracion_computable NUMERIC(10,2) DEFAULT 0,
    meses_computables INT DEFAULT 0,
    dias_faltas INT DEFAULT 0,
    monto_descuento_faltas NUMERIC(10,2) DEFAULT 0,
    monto_cts NUMERIC(10,2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 3. Tabla para liquidaciones de cese (Ley 30057 y DL 276)
CREATE TABLE liquidaciones_cese (
    id SERIAL PRIMARY KEY,
    tenant_id INT NOT NULL,
    contrato_id INT NOT NULL REFERENCES contratos(id) ON DELETE CASCADE,
    fecha_inicio_computable DATE NOT NULL,
    fecha_cese DATE NOT NULL,
    motivo VARCHAR(100),
    anos_servicios INT DEFAULT 0,
    meses_servicios INT DEFAULT 0,
    remuneracion_computable NUMERIC(10,2) DEFAULT 0,
    monto_cts NUMERIC(10,2) DEFAULT 0,
    monto_vacaciones_truncas NUMERIC(10,2) DEFAULT 0,
    monto_gratificacion_trunca NUMERIC(10,2) DEFAULT 0,
    total_liquidacion NUMERIC(10,2) DEFAULT 0,
    estado VARCHAR(20) DEFAULT 'BORRADOR',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS liquidaciones_cese;
DROP TABLE IF EXISTS planilla_cts_detalles;
DROP TABLE IF EXISTS planillas_cts;
-- +goose StatementEnd
