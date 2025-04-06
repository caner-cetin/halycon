package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/caner-cetin/halycon/internal"
	"github.com/rs/zerolog/log"
)

func SetDefaultMerchant() error {
	if len(Config.Amazon.Auth.Merchants) == 0 {
		return fmt.Errorf("no merchants configured")
	}
	var defaultMerchantSet bool
	for _, merchant := range Config.Amazon.Auth.Merchants {
		if merchant.RefreshToken == "" {
			return fmt.Errorf("merchant refresh token not set")
		}
		if merchant.SellerToken == "" {
			return fmt.Errorf("merchant seller token not set")
		}
		if len(merchant.MarketplaceID) == 0 {
			merchant.MarketplaceID = []string{"ATVPDKIKX0DER"}
		}
		if merchant.Default {
			if defaultMerchantSet {
				return fmt.Errorf("more than one default merchants specified")
			}
			Config.Amazon.Auth.DefaultMerchant = merchant
			defaultMerchantSet = true
		}
	}
	if !defaultMerchantSet {
		log.Warn().Msg("default merchant not set")
		shouldSet, err := internal.PromptFor("set default merchant? [Y/n]")
		if err != nil {
			return fmt.Errorf("failed to prompt for default merchant")
		}
		if strings.EqualFold(shouldSet, "y") || shouldSet == "" {
			for i, merchant := range Config.Amazon.Auth.Merchants {
				if merchant.Name != "" {
					fmt.Printf("%d) %s \n", i, merchant.Name)
				} else {
					fmt.Printf("%d) %s \n", i, merchant.SellerToken)
				}
			}
			Config.Amazon.Auth.DefaultMerchant, Config.Amazon.Auth.DefaultMerchantIndex, err = internal.PromptForPickFromSlice("choose default: (0,1,2,3...) ", Config.Amazon.Auth.Merchants)
			if err != nil {
				return fmt.Errorf("failed to select default merchant")
			}
			Config.Amazon.Auth.Merchants[Config.Amazon.Auth.DefaultMerchantIndex].Default = true
		} else {
			return fmt.Errorf("default merchant required")
		}
	}
	return nil
}

func SetDefaultShipFromAddress() error {
	if Config.Amazon.FBA.Enabled {
		if len(Config.Amazon.FBA.ShipFrom) == 0 {
			return fmt.Errorf("no ship-from addresses configured")
		}
		var defaultAddressSet bool
		for _, address := range Config.Amazon.FBA.ShipFrom {
			if address.AddressLine1 == "" {
				return fmt.Errorf("address line 1 not set")
			}
			if address.City == "" {
				return fmt.Errorf("city not set")
			}
			if address.Name == "" {
				return fmt.Errorf("primary contact name not set")
			}
			if address.PhoneNumber == "" {
				return fmt.Errorf("phone number not set")
			}
			if address.PostalCode == "" {
				return fmt.Errorf("postal code not set")
			}
			if address.CountryCode == "" {
				address.CountryCode = "US"
			}
			if address.Default {
				if defaultAddressSet {
					return fmt.Errorf("more than one default addresses specified")
				}
				Config.Amazon.FBA.DefaultShipFrom = address
				defaultAddressSet = true
			}
		}
		if !defaultAddressSet {
			log.Warn().Msg("default address not set")
			shouldSet, err := internal.PromptFor("set default address? [Y/n]")
			if err != nil {
				return fmt.Errorf("failed to prompt for default address")
			}
			if strings.EqualFold(shouldSet, "y") || shouldSet == "" {
				for i, address := range Config.Amazon.FBA.ShipFrom {
					fmt.Printf("%d) %s %s \n", i, address.AddressLine1, address.Name)
				}
				Config.Amazon.FBA.DefaultShipFrom, Config.Amazon.FBA.DefaultShipFromIndex, err = internal.PromptForPickFromSlice("choose default: (0,1,2,3...) ", Config.Amazon.FBA.ShipFrom)
				if err != nil {
					return fmt.Errorf("failed to select default address")
				}
				Config.Amazon.FBA.ShipFrom[Config.Amazon.FBA.DefaultShipFromIndex].Default = true
			} else {
				return fmt.Errorf("default address required")
			}
		}
	}
	return nil
}

func SetDefaultClient() error {
	if len(Config.Amazon.Auth.Clients) == 0 {
		return fmt.Errorf("no clients configured")
	}
	var defaultClientSet bool
	for _, client := range Config.Amazon.Auth.Clients {
		if client.ID == "" {
			return fmt.Errorf("client id not set")
		}
		if client.Secret == "" {
			return fmt.Errorf("client secret not set")
		}
		if client.AuthEndpoint == "" {
			client.AuthEndpoint = "https://api.amazon.com/auth/o2/token"
		}
		if client.APIEndpoint == "" {
			client.APIEndpoint = "sellingpartnerapi-na.amazon.com"
		}
		if client.Default {
			if defaultClientSet {
				return fmt.Errorf("more than one default client set")
			}
			Config.Amazon.Auth.DefaultClient = client
			defaultClientSet = true
		}
	}
	if !defaultClientSet {
		log.Warn().Msg("default client not set")
		shouldSet, err := internal.PromptFor("set default client? [Y/n]")
		if err != nil {
			return fmt.Errorf("failed to prompt for default client")
		}
		if strings.EqualFold(shouldSet, "y") || shouldSet == "" {
			for i, client := range Config.Amazon.Auth.Clients {
				if client.Name != "" {
					fmt.Printf("%d) %s \n", i, client.Name)
				} else {
					fmt.Printf("%d) %s \n", i, client.ID)
				}
			}
			Config.Amazon.Auth.DefaultClient, Config.Amazon.Auth.DefaultClientIndex, err = internal.PromptForPickFromSlice("choose default: (0,1,2,3...) ", Config.Amazon.Auth.Clients)
			if err != nil {
				return fmt.Errorf("failed to select default client")
			}
			Config.Amazon.Auth.Clients[Config.Amazon.Auth.DefaultClientIndex].Default = true
		} else {
			return fmt.Errorf("default client required")
		}
	}
	return nil
}

func SetOtherDefaults() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	Config.Amazon.DefaultLanguageTag = "en_US"
	Config.Sqlite.Path = filepath.Join(home, ".halycon.db")
	return nil
}
