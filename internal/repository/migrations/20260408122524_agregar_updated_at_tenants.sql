-- +goose Up
-- +goose StatementBegin
ALTER TABLE tenants 
ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenants 
DROP COLUMN updated_at;
-- +goose StatementEnd
