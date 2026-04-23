-- +goose Up
CREATE TABLE ocurrencias_asistencia (
    id SERIAL PRIMARY KEY,
    contrato_id INTEGER NOT NULL REFERENCES contratos(id),
    tipo VARCHAR(20) NOT NULL, -- 'INASISTENCIA' o 'TARDANZA'
    fecha_ocurrencia DATE NOT NULL,
    cantidad NUMERIC(10,2) NOT NULL, -- Días para inasistencia, Minutos para tardanza
    procesado BOOLEAN DEFAULT false, -- ¡LA CLAVE!
    planilla_id_descuento INTEGER, -- En qué planilla finalmente se le descontó
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE ocurrencias_asistencia;
