-- name: GetAsinToSkuMapContents :many
select sku,
  asin,
  title
from fba_inventory;
-- name: GetFBAProductFromAsin :one
select *
from fba_inventory
where asin = ?;
-- name: FbaInventoryCount :one
select COUNT(sku)
from fba_inventory
group by sku;
