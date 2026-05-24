-- Migración: Ampliar límite de descripción de metas presupuestales a 512 caracteres
-- Creado: 2026-05-24

-- Up
ALTER TABLE metas_presupuestales ALTER COLUMN descripcion TYPE character varying(512);

-- Down
-- ALTER TABLE metas_presupuestales ALTER COLUMN descripcion TYPE character varying(255);
