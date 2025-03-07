package config

type Config struct {
	Key     string
	Default string
}

var (
	AMAZON_AUTH_CLIENT_ID               = Config{Key: "amazon.auth.client.id"}
	AMAZON_AUTH_CLIENT_SECRET           = Config{Key: "amazon.auth.client.secret"}
	AMAZON_AUTH_REFRESH_TOKEN           = Config{Key: "amazon.auth.refresh_token"}
	AMAZON_AUTH_ENDPOINT                = Config{Key: "amazon.auth.endpoint", Default: "https://api.amazon.com/auth/o2/token"}
	AMAZON_AUTH_HOST                    = Config{Key: "amazon.auth.host", Default: "sellingpartnerapi-na.amazon.com"}
	AMAZON_MARKETPLACE_ID               = Config{Key: "amazon.marketplace_id", Default: "ATVPDKIKX0DER"}
	AMAZON_FBA_SHIP_FROM_ADDRESS_LINE_1 = Config{Key: "amazon.fba.ship_from.address_line_1"}
	AMAZON_FBA_SHIP_FROM_ADDRESS_LINE_2 = Config{Key: "amazon.fba.ship_from.address_line_2"}
	AMAZON_FBA_SHIP_FROM_CITY           = Config{Key: "amazon.fba.ship_from.city"}
	AMAZON_FBA_SHIP_FROM_COMPANY_NAME   = Config{Key: "amazon.fba.ship_from.company_name"}
	AMAZON_FBA_SHIP_FROM_COUNTRY_CODE   = Config{Key: "amazon.fba.ship_from.country_code", Default: "US"}
	AMAZON_FBA_SHIP_FROM_EMAIL          = Config{Key: "amazon.fba.ship_from.email"}
	AMAZON_FBA_SHIP_FROM_NAME           = Config{Key: "amazon.fba.ship_from.name"}
	AMAZON_FBA_SHIP_FROM_PHONE_NUMBER   = Config{Key: "amazon.fba.ship_from.phone_number"}
	AMAZON_FBA_SHIP_FROM_POSTAL_CODE    = Config{Key: "amazon.fba.ship_from.postal_code"}
	AMAZON_FBA_SHIP_FROM_STATE_PROVINCE = Config{Key: "amazon.fba.ship_from.state_or_province_code"}
)
