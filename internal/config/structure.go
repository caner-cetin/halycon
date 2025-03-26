package config

// mapstructure tag is for https://pkg.go.dev/github.com/spf13/viper
// yaml is for https://pkg.go.dev/gopkg.in/yaml.v3

type Cfg struct {
	Amazon AmazonConfig `mapstructure:"amazon" yaml:"amazon"`
	Groq   GroqConfig   `mapstructure:"groq" yaml:"groq"`
	Path   string       `yaml:"-"`
}

type GroqConfig struct {
	Token string `mapstructure:"token" yaml:"token"`
}

type AmazonConfig struct {
	Auth               AuthConfig `mapstructure:"auth" yaml:"auth"`
	FBA                FBAConfig  `mapstructure:"fba" yaml:"fba"`
	DefaultLanguageTag string     `mapstructure:"default_language_tag" yaml:"default_language_tag"`
}

type AuthConfig struct {
	DefaultClient        ClientConfig     `yaml:"-"`
	DefaultClientIndex   int              `yaml:"-"`
	DefaultMerchant      MerchantConfig   `yaml:"-"`
	DefaultMerchantIndex int              `yaml:"-"`
	Clients              []ClientConfig   `mapstructure:"clients" yaml:"clients"`
	Merchants            []MerchantConfig `mapstructure:"merchants" yaml:"merchants"`
}

type ClientConfig struct {
	// ID get this value when you register your application.
	// Refer to https://developer-docs.amazon.com/sp-api/docs/viewing-your-application-information-and-credentials
	ID string `mapstructure:"id" yaml:"id"`
	// Name of client, not used for anything Amazon related, only for referencing the client throughout the code.
	// If not specified, client will be referenced by its id.
	Name string `mapstructure:"name" yaml:"name"`
	// Secret get this value when you register your application.
	// Refer to https://developer-docs.amazon.com/sp-api/docs/viewing-your-application-information-and-credentials
	Secret string `mapstructure:"secret" yaml:"secret"`
	// AuthEndpoint is uh...
	AuthEndpoint string `mapstructure:"auth_endpoint" yaml:"auth_endpoint"`
	// APIEndpoint is the Selling Partner API endpoint, see https://developer-docs.amazon.com/sp-api/docs/sp-api-endpoints
	// Omit the scheme, like this: sellingpartnerapi-na.amazon.com
	APIEndpoint string `mapstructure:"api_endpoint" yaml:"api_endpoint"`
	// Default client configuration to use
	Default bool `mapstructure:"default" yaml:"default"`
}

// FBAConfig holds the configuration for Fulfillment By Amazon (FBA) settings
type FBAConfig struct {
	Enabled              bool             `mapstructure:"enabled" yaml:"enabled"`
	DefaultShipFrom      ShipFromConfig   `yaml:"-"`
	DefaultShipFromIndex int              `yaml:"-"`
	ShipFrom             []ShipFromConfig `mapstructure:"ship_from" yaml:"ship_from"`
}

// ShipFromConfig holds the address and contact information for FBA shipping origin
type ShipFromConfig struct {
	// AddressLine1 is street address information.
	AddressLine1 string `mapstructure:"address_line_1" yaml:"address_line_1"`
	// AddressLine2 is additional street address information.
	AddressLine2 string `mapstructure:"address_line_2" yaml:"address_line_2"`
	City         string `mapstructure:"city" yaml:"city"`
	// CompanyName is the name of the business.
	CompanyName string `mapstructure:"company_name" yaml:"company_name"`
	// CountryCode is the country code in two-character ISO 3166-1 alpha-2 format.
	CountryCode string `mapstructure:"country_code" yaml:"country_code"`
	Email       string `mapstructure:"email" yaml:"email"`
	// Name is the name of the individual who is the primary contact.
	Name            string `mapstructure:"name" yaml:"name"`
	PhoneNumber     string `mapstructure:"phone_number" yaml:"phone_number"`
	PostalCode      string `mapstructure:"postal_code" yaml:"postal_code"`
	StateOrProvince string `mapstructure:"state_or_province_code" yaml:"state_or_province_code"`
	Default         bool   `mapstructure:"default" yaml:"default"`
}

// MerchantConfig holds the authentication and marketplace configuration for Amazon merchants
type MerchantConfig struct {
	// RefreshToken acquired from self-authorizing the application
	RefreshToken string `mapstructure:"refresh_token" yaml:"refresh_token"`
	// Default refresh token or not
	Default bool `mapstructure:"default" yaml:"default"`
	// SellerToken ?
	SellerToken string `mapstructure:"seller_token" yaml:"seller_token"`
	// MarketplaceID for all kinds of operations, US will be registered by default
	MarketplaceID []string `mapstructure:"marketplace_id" yaml:"marketplace_id"`
	// Name of merchant, not used for anything Amazon related, only for referencing the client throughout the code.
	// If not specified, merchant will be referenced by its seller token.
	Name string `mapstructure:"name" yaml:"name"`
}

var Config Cfg
