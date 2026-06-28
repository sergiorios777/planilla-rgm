-- +goose Up
-- +goose StatementBegin
ALTER TABLE liquidaciones_cese 
  ADD COLUMN monto_vacaciones_no_gozadas NUMERIC(10,2) DEFAULT 0,
  ADD COLUMN monto_indemnizacion_vacacional NUMERIC(10,2) DEFAULT 0,
  ADD COLUMN periodos_vencidos_vacaciones INT DEFAULT 0,
  ADD COLUMN periodos_no_vencidos_vacaciones INT DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE liquidaciones_cese 
  DROP COLUMN IF EXISTS monto_vacaciones_no_gozadas,
  DROP COLUMN IF EXISTS monto_indemnizacion_vacacional,
  DROP COLUMN IF EXISTS periodos_vencidos_vacaciones,
  DROP COLUMN IF EXISTS periodos_no_vencidos_vacaciones;
-- +goose StatementEnd
