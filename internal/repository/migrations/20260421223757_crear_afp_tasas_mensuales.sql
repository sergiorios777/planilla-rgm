-- +goose Up
-- 1. Tabla de Tasas Históricas de AFP (Por mes)
CREATE TABLE afp_tasas_mensuales (
    id SERIAL PRIMARY KEY,
    afp_id INTEGER REFERENCES afps(id),
    anio INTEGER NOT NULL,
    mes INTEGER NOT NULL,
    aporte_obligatorio NUMERIC(5,4) DEFAULT 0.1000, -- 10.00%
    comision_flujo NUMERIC(5,4) NOT NULL,           -- Ej: 0.0155 (1.55%)
    comision_mixta_flujo NUMERIC(5,4) NOT NULL,     -- Ej: 0.0000 o 0.0023
    prima_seguro NUMERIC(5,4) NOT NULL,             -- Ej: 0.0184 (1.84%)
    UNIQUE (afp_id, anio, mes)
);

-- +goose Down
-- 1. Eliminar la tabla afp_tasas_mensuales
DROP TABLE IF EXISTS afp_tasas_mensuales;
