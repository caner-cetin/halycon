-- +goose Up
-- +goose StatementBegin
CREATE VIRTUAL TABLE fts_title_quantity USING FTS5(
  title,
  total_quantity,
  fulfillable_quantity,
  inbound_receiving_quantity,
  inbound_shipped_quantity
);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS fts_title_quantity;
-- +goose StatementEnd
