-- +goose Up
-- +goose StatementBegin
CREATE TABLE fba_inventory (
  title TEXT,
  total_quantity INTEGER,
  fulfillable_quantity INTEGER,
  inbound_receiving_quantity INTEGER,
  inbound_shipped_quantity INTEGER,
  sku TEXT,
  asin TEXT
);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS fba_inventory;
-- +goose StatementEnd
