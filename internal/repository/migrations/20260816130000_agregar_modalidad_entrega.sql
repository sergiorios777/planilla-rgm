-- +goose Up
ALTER TABLE conceptos_modelo 
  ADD COLUMN modalidad_entrega VARCHAR(20) NOT NULL DEFAULT 'PERMANENTE'
  CHECK (modalidad_entrega IN ('PERMANENTE', 'PERIODICO', 'EXCEPCIONAL', 'OCASIONAL'));

ALTER TABLE conceptos_tenant 
  ADD COLUMN modalidad_entrega VARCHAR(20) NOT NULL DEFAULT 'PERMANENTE'
  CHECK (modalidad_entrega IN ('PERMANENTE', 'PERIODICO', 'EXCEPCIONAL', 'OCASIONAL'));

-- Migrar datos existentes en conceptos_modelo
UPDATE conceptos_modelo SET modalidad_entrega = 'OCASIONAL' WHERE es_ocasional = true;
UPDATE conceptos_modelo SET modalidad_entrega = 'EXCEPCIONAL' WHERE es_extraordinario = true AND es_ocasional = false;
UPDATE conceptos_modelo SET modalidad_entrega = 'PERIODICO' WHERE frecuencia_meses NOT IN ('1,2,3,4,5,6,7,8,9,10,11,12', '') AND es_ocasional = false AND es_extraordinario = false;

-- Migrar datos existentes en conceptos_tenant
UPDATE conceptos_tenant SET modalidad_entrega = 'OCASIONAL' WHERE es_ocasional = true;
UPDATE conceptos_tenant SET modalidad_entrega = 'EXCEPCIONAL' WHERE es_extraordinario = true AND es_ocasional = false;
UPDATE conceptos_tenant SET modalidad_entrega = 'PERIODICO' WHERE frecuencia_meses NOT IN ('1,2,3,4,5,6,7,8,9,10,11,12', '') AND es_ocasional = false AND es_extraordinario = false;

-- +goose Down
ALTER TABLE conceptos_tenant DROP COLUMN IF EXISTS modalidad_entrega;
ALTER TABLE conceptos_modelo DROP COLUMN IF EXISTS modalidad_entrega;
