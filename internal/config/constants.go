package config

// Config is a struct that represents a configuration key-value pair.
// It stores both the key identifier and its default value.
type Config struct {
	// Key is the identifier for the configuration setting
	Key string
	// Default is the value to be used if no override is provided
	Default string
}

var (
	// AMAZON_AUTH_CLIENT_ID get this value when you register your application.
	// Refer to https://developer-docs.amazon.com/sp-api/docs/viewing-your-application-information-and-credentials
	AMAZON_AUTH_CLIENT_ID = Config{Key: "amazon.auth.client.id"}
	// AMAZON_AUTH_CLIENT_SECRET get this value when you register your application.
	// Refer to https://developer-docs.amazon.com/sp-api/docs/viewing-your-application-information-and-credentials
	AMAZON_AUTH_CLIENT_SECRET = Config{Key: "amazon.auth.client.secret"}
	// AMAZON_AUTH_REFRESH_TOKEN is the LWA refresh token. Get this value when the selling partner authorizes your application.
	// For more information, refer to https://developer-docs.amazon.com/sp-api/docs/authorizing-selling-partner-api-applications
	AMAZON_AUTH_REFRESH_TOKEN = Config{Key: "amazon.auth.refresh_token"}
	// AMAZON_AUTH_ENDPOINT is ??? but it works lol.
	AMAZON_AUTH_ENDPOINT = Config{Key: "amazon.auth.endpoint", Default: "https://api.amazon.com/auth/o2/token"}
	// AMAZON_AUTH_HOST is the Selling Partner API endpoint, see https://developer-docs.amazon.com/sp-api/docs/sp-api-endpoints
	// Omit the scheme.
	AMAZON_AUTH_HOST = Config{Key: "amazon.auth.host", Default: "sellingpartnerapi-na.amazon.com"}
	// AMAZON_MARKETPLACE_ID is list of identifiers that represent each marketplace.
	// Refer to https://developer-docs.amazon.com/sp-api/docs/marketplace-ids#
	AMAZON_MARKETPLACE_ID = Config{Key: "amazon.marketplace_id", Default: "ATVPDKIKX0DER"}

	// https://developer-docs.amazon.com/sp-api/docs/fulfillment-inbound-api-v2024-03-20-reference#addressinput

	// AMAZON_FBA_SHIP_FROM_ADDRESS_LINE_1 is street address information.
	AMAZON_FBA_SHIP_FROM_ADDRESS_LINE_1 = Config{Key: "amazon.fba.ship_from.address_line_1"}
	// AMAZON_FBA_SHIP_FROM_ADDRESS_LINE_2 is additional street address information.
	AMAZON_FBA_SHIP_FROM_ADDRESS_LINE_2 = Config{Key: "amazon.fba.ship_from.address_line_2"}
	// AMAZON_FBA_SHIP_FROM_CITY is... you guessed it. The city.
	AMAZON_FBA_SHIP_FROM_CITY = Config{Key: "amazon.fba.ship_from.city"}
	// AMAZON_FBA_SHIP_FROM_COMPANY_NAME is the name of the business.
	AMAZON_FBA_SHIP_FROM_COMPANY_NAME = Config{Key: "amazon.fba.ship_from.company_name"}
	// AMAZON_FBA_SHIP_FROM_COUNTRY_CODE is the country code in two-character ISO 3166-1 alpha-2 format.
	AMAZON_FBA_SHIP_FROM_COUNTRY_CODE = Config{Key: "amazon.fba.ship_from.country_code", Default: "US"}
	// AMAZON_FBA_SHIP_FROM_EMAIL is... the email address.
	AMAZON_FBA_SHIP_FROM_EMAIL = Config{Key: "amazon.fba.ship_from.email"}
	// AMAZON_FBA_SHIP_FROM_NAME is the name of the individual who is the primary contact.
	AMAZON_FBA_SHIP_FROM_NAME = Config{Key: "amazon.fba.ship_from.name"}
	// AMAZON_FBA_SHIP_FROM_PHONE_NUMBER is... take a gueeesss!
	AMAZON_FBA_SHIP_FROM_PHONE_NUMBER = Config{Key: "amazon.fba.ship_from.phone_number"}
	// AMAZON_FBA_SHIP_FROM_POSTAL_CODE is... can you find what it is? Mhm! The postal code!
	AMAZON_FBA_SHIP_FROM_POSTAL_CODE = Config{Key: "amazon.fba.ship_from.postal_code"}
	// AMAZON_FBA_SHIP_FROM_STATE_PROVINCE is the state or province code.
	AMAZON_FBA_SHIP_FROM_STATE_PROVINCE = Config{Key: "amazon.fba.ship_from.state_or_province_code"}
	// AMAZON_MERCHANT_TOKEN can be acquired from https://sellercentral.amazon.com/sw/AccountInfo/MerchantToken
	AMAZON_MERCHANT_TOKEN = Config{Key: "amazon.merchant_token"}
	// AMAZON_DEFAULT_LANGUAGE_TAG is the tag, defaults to en_US.
	AMAZON_DEFAULT_LANGUAGE_TAG = Config{Key: "amazon.default_language_tag", Default: "en_US"}
)
