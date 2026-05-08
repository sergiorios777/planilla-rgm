-- +goose Up
-- +goose StatementBegin

CREATE TABLE conceptos_modelo (
    id SERIAL PRIMARY KEY,
    regimen_id INTEGER NOT NULL REFERENCES regimenes_laborales(id),
    concepto_id INTEGER NOT NULL REFERENCES conceptos_maestros(id),
    nombre_personalizado VARCHAR(150) NOT NULL,
    frecuencia_meses VARCHAR(50) DEFAULT '1,2,3,4,5,6,7,8,9,10,11,12',
    clasificador_id INTEGER REFERENCES clasificadores_mef(id) ON DELETE SET NULL,
    es_extraordinario BOOLEAN DEFAULT false,
    requiere_monto BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(regimen_id, nombre_personalizado) -- Evita registrar el mismo concepto dos veces en un mismo régimen
);

-- Modificacion de la tabla conceptos_tenant para agregar el campo requiere_monto
ALTER TABLE conceptos_tenant 
ADD COLUMN requiere_monto BOOLEAN DEFAULT false;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE conceptos_modelo;

-- Modificacion de la tabla conceptos_tenant para eliminar el campo requiere_monto
ALTER TABLE conceptos_tenant 
DROP COLUMN requiere_monto;

-- +goose StatementEnd
