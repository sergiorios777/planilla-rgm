-- +goose Up
ALTER TABLE planilla_conceptos 
ADD COLUMN maestro_id integer NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE planilla_conceptos 
DROP COLUMN maestro_id;
