-- +goose Up
-- +goose StatementBegin
-- Drop and recreate FTS table with UPC column
DROP TABLE IF EXISTS fts_title_quantity;
CREATE VIRTUAL TABLE fts_title_quantity USING FTS5(
  title,
  total_quantity,
  fulfillable_quantity,
  inbound_receiving_quantity,
  inbound_shipped_quantity,
  upc
);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
-- Recreate FTS table without UPC column
DROP TABLE IF EXISTS fts_title_quantity;
CREATE VIRTUAL TABLE fts_title_quantity USING FTS5(
  title,
  total_quantity,
  fulfillable_quantity,
  inbound_receiving_quantity,
  inbound_shipped_quantity
);
-- +goose StatementEnd