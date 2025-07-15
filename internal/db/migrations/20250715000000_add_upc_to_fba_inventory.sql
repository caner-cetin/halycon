-- +goose Up
-- +goose StatementBegin
ALTER TABLE fba_inventory ADD COLUMN upc TEXT;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
ALTER TABLE fba_inventory DROP COLUMN upc;
-- +goose StatementEnd