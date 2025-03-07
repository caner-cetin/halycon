package internal

const (
	CONFIG_KEY_AMAZON_AUTH_CLIENT_ID     = "amazon.auth.client.id"
	CONFIG_KEY_AMAZON_AUTH_CLIENT_SECRET = "amazon.auth.client.secret"
	CONFIG_KEY_AMAZON_AUTH_REFRESH_TOKEN = "amazon.auth.refresh_token"
	CONFIG_KEY_AMAZON_AUTH_ENDPOINT      = "amazon.auth.endpoint"
	CONFIG_KEY_AMAZON_AUTH_HOST          = "amazon.auth.host"
	CONFIG_KEY_AMAZON_MARKETPLACE_ID     = "amazon.marketplace_id"
)

const (
	DEFAULT_AMAZON_AUTH_ENDPOINT  = "https://api.amazon.com/auth/o2/token"
	DEFAULT_AMAZON_AUTH_HOST      = "sellingpartnerapi-na.amazon.com"
	DEFAULT_AMAZON_MARKETPLACE_ID = "ATVPDKIKX0DER"
)

type ContextKey string

const (
	APP_CONTEXT ContextKey = "halycon.ctx"
)
