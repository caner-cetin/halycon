package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/caner-cetin/halycon/internal/config"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Interactive configuration generator",
	Long:  "Generate a configuration file interactively with prompts for all required and optional settings",
	Run:   generateConfig,
}

func getConfigCmd() *cobra.Command {
	return configCmd
}

func generateConfig(cmd *cobra.Command, args []string) {
	var newConfig config.Cfg
	
	var configPath string
	if cfgFile != "" {
		configPath = cfgFile
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Error().Err(err).Msg("failed to get user home directory")
			return
		}
		configPath = filepath.Join(home, ".halycon.yaml")
	}
	
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		MarginBottom(1)
	
	fmt.Printf("%s\n", headerStyle.Render("Halycon Configuration Generator"))
	fmt.Printf("Creating configuration file at: %s\n\n", configPath)
	
	if err := configureAmazonBasics(&newConfig); err != nil {
		log.Error().Err(err).Msg("failed to configure Amazon basics")
		return
	}
	
	if err := configureClients(&newConfig); err != nil {
		log.Error().Err(err).Msg("failed to configure clients")
		return
	}
	
	if err := configureMerchants(&newConfig); err != nil {
		log.Error().Err(err).Msg("failed to configure merchants")
		return
	}
	
	if err := configureFBA(&newConfig); err != nil {
		log.Error().Err(err).Msg("failed to configure FBA")
		return
	}
	
	if err := configureOptionalServices(&newConfig); err != nil {
		log.Error().Err(err).Msg("failed to configure optional services")
		return
	}
	
	yamlData, err := yaml.Marshal(newConfig)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal config to yaml")
		return
	}
	
	if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
		log.Error().Err(err).Msg("failed to write config file")
		return
	}
	
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#22C55E")).
		Bold(true)
	
	fmt.Printf("\n%s\n", successStyle.Render("✅ Configuration generated successfully!"))
	fmt.Printf("📁 Config file: %s\n", configPath)
	fmt.Println("\n🎉 You can now use other halycon commands with your configuration.")
	
	var viewConfig bool
	huh.NewConfirm().
		Title("View generated configuration?").
		Value(&viewConfig).
		Run()
	
	if viewConfig {
		fmt.Printf("\n--- Generated Configuration ---\n%s\n", string(yamlData))
	}
}

func configureAmazonBasics(newConfig *config.Cfg) error {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B82F6")).
		Bold(true).
		MarginTop(1)
	
	fmt.Printf("%s\n", headerStyle.Render("🔧 Amazon Selling Partner API Configuration"))
	
	var languageTag string
	err := huh.NewInput().
		Title("Default Language Tag").
		Description("Language tag for Amazon API requests (e.g., en_US, de_DE)").
		Value(&languageTag).
		Placeholder("en_US").
		Run()
	
	if err != nil {
		return err
	}
	
	if languageTag == "" {
		languageTag = "en_US"
	}
	newConfig.Amazon.DefaultLanguageTag = languageTag
	
	return nil
}

func configureClients(newConfig *config.Cfg) error {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B82F6")).
		Bold(true).
		MarginTop(1)
	
	fmt.Printf("%s\n", headerStyle.Render("🔑 Amazon API Clients"))
	
	var clientCountStr string
	err := huh.NewInput().
		Title("Number of clients").
		Description("How many Amazon API clients do you want to configure?").
		Value(&clientCountStr).
		Placeholder("1").
		Validate(func(s string) error {
			if s == "" {
				return nil
			}
			if _, err := strconv.Atoi(s); err != nil {
				return fmt.Errorf("must be a number")
			}
			return nil
		}).
		Run()
	
	if err != nil {
		return err
	}
	
	if clientCountStr == "" {
		clientCountStr = "1"
	}
	
	clientCount, err := strconv.Atoi(clientCountStr)
	if err != nil {
		return err
	}
	
	for i := 0; i < clientCount; i++ {
		var client config.ClientConfig
		
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fmt.Sprintf("Client %d - ID", i+1)).
					Description("Your Amazon API Client ID (required)").
					Value(&client.ID).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("client ID is required")
						}
						return nil
					}),
				
				huh.NewInput().
					Title("Client Secret").
					Description("Your Amazon API Client Secret (required)").
					Value(&client.Secret).
					EchoMode(huh.EchoModePassword).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("client secret is required")
						}
						return nil
					}),
				
				huh.NewInput().
					Title("Client Name").
					Description("Friendly name for this client (optional)").
					Value(&client.Name).
					Placeholder("My Amazon Client"),
				
				huh.NewInput().
					Title("Auth Endpoint").
					Description("OAuth2 token endpoint").
					Value(&client.AuthEndpoint).
					Placeholder("https://api.amazon.com/auth/o2/token"),
				
				huh.NewInput().
					Title("API Endpoint").
					Description("Selling Partner API endpoint (without https://)").
					Value(&client.APIEndpoint).
					Placeholder("sellingpartnerapi-na.amazon.com"),
			),
		)
		
		if err := form.Run(); err != nil {
			return err
		}
		
		if client.AuthEndpoint == "" {
			client.AuthEndpoint = "https://api.amazon.com/auth/o2/token"
		}
		if client.APIEndpoint == "" {
			client.APIEndpoint = "sellingpartnerapi-na.amazon.com"
		}
		
		if i == 0 {
			client.Default = true
		} else {
			huh.NewConfirm().
				Title("Make this the default client?").
				Value(&client.Default).
				Run()
		}
		
		newConfig.Amazon.Auth.Clients = append(newConfig.Amazon.Auth.Clients, client)
	}
	
	return nil
}

func configureMerchants(newConfig *config.Cfg) error {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B82F6")).
		Bold(true).
		MarginTop(1)
	
	fmt.Printf("%s\n", headerStyle.Render("🏪 Amazon Merchants"))
	
	var merchantCountStr string
	err := huh.NewInput().
		Title("Number of merchants").
		Description("How many Amazon merchant accounts do you want to configure?").
		Value(&merchantCountStr).
		Placeholder("1").
		Validate(func(s string) error {
			if s == "" {
				return nil
			}
			if _, err := strconv.Atoi(s); err != nil {
				return fmt.Errorf("must be a number")
			}
			return nil
		}).
		Run()
	
	if err != nil {
		return err
	}
	
	if merchantCountStr == "" {
		merchantCountStr = "1"
	}
	
	merchantCount, err := strconv.Atoi(merchantCountStr)
	if err != nil {
		return err
	}
	
	for i := 0; i < merchantCount; i++ {
		var merchant config.MerchantConfig
		var marketplaceIDs string
		
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fmt.Sprintf("Merchant %d - Refresh Token", i+1)).
					Description("Your Amazon refresh token (required)").
					Value(&merchant.RefreshToken).
					EchoMode(huh.EchoModePassword).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("refresh token is required")
						}
						return nil
					}),
				
				huh.NewInput().
					Title("Seller Token").
					Description("Your Amazon seller token (required)").
					Value(&merchant.SellerToken).
					EchoMode(huh.EchoModePassword).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("seller token is required")
						}
						return nil
					}),
				
				huh.NewInput().
					Title("Merchant Name").
					Description("Friendly name for this merchant (optional)").
					Value(&merchant.Name).
					Placeholder("My Amazon Store"),
				
				huh.NewInput().
					Title("Marketplace IDs").
					Description("Comma-separated marketplace IDs").
					Value(&marketplaceIDs).
					Placeholder("ATVPDKIKX0DER (US)"),
			),
		)
		
		if err := form.Run(); err != nil {
			return err
		}
		
		if marketplaceIDs == "" {
			merchant.MarketplaceID = []string{"ATVPDKIKX0DER"}
		} else {
			merchant.MarketplaceID = strings.Split(marketplaceIDs, ",")
			for j := range merchant.MarketplaceID {
				merchant.MarketplaceID[j] = strings.TrimSpace(merchant.MarketplaceID[j])
			}
		}
		
		if i == 0 {
			merchant.Default = true
		} else {
			huh.NewConfirm().
				Title("Make this the default merchant?").
				Value(&merchant.Default).
				Run()
		}
		
		newConfig.Amazon.Auth.Merchants = append(newConfig.Amazon.Auth.Merchants, merchant)
	}
	
	return nil
}

func configureFBA(newConfig *config.Cfg) error {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B82F6")).
		Bold(true).
		MarginTop(1)
	
	fmt.Printf("%s\n", headerStyle.Render("📦 FBA Configuration"))
	
	var fbaEnabled bool
	err := huh.NewConfirm().
		Title("Enable FBA features?").
		Description("Enable Fulfillment by Amazon functionality").
		Value(&fbaEnabled).
		Run()
	
	if err != nil {
		return err
	}
	
	newConfig.Amazon.FBA.Enabled = fbaEnabled
	
	if !fbaEnabled {
		return nil
	}
	
	var shipFromCountStr string
	err = huh.NewInput().
		Title("Number of ship-from addresses").
		Description("How many FBA ship-from addresses do you want to configure?").
		Value(&shipFromCountStr).
		Placeholder("1").
		Validate(func(s string) error {
			if s == "" {
				return nil
			}
			if _, err := strconv.Atoi(s); err != nil {
				return fmt.Errorf("must be a number")
			}
			return nil
		}).
		Run()
	
	if err != nil {
		return err
	}
	
	if shipFromCountStr == "" {
		shipFromCountStr = "1"
	}
	
	shipFromCount, err := strconv.Atoi(shipFromCountStr)
	if err != nil {
		return err
	}
	
	for i := 0; i < shipFromCount; i++ {
		var shipFrom config.ShipFromConfig
		
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fmt.Sprintf("Ship-From Address %d - Address Line 1", i+1)).
					Description("Street address (required)").
					Value(&shipFrom.AddressLine1).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("address line 1 is required")
						}
						return nil
					}),
				
				huh.NewInput().
					Title("Address Line 2").
					Description("Apartment, suite, etc. (optional)").
					Value(&shipFrom.AddressLine2),
				
				huh.NewInput().
					Title("City").
					Description("City name (required)").
					Value(&shipFrom.City).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("city is required")
						}
						return nil
					}),
				
				huh.NewInput().
					Title("State/Province").
					Description("State or province code (required)").
					Value(&shipFrom.StateOrProvince).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("state/province is required")
						}
						return nil
					}),
				
				huh.NewInput().
					Title("Postal Code").
					Description("ZIP or postal code (required)").
					Value(&shipFrom.PostalCode).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("postal code is required")
						}
						return nil
					}),
			),
			
			huh.NewGroup(
				huh.NewInput().
					Title("Country Code").
					Description("Two-letter country code").
					Value(&shipFrom.CountryCode).
					Placeholder("US"),
				
				huh.NewInput().
					Title("Company Name").
					Description("Business name (optional)").
					Value(&shipFrom.CompanyName),
				
				huh.NewInput().
					Title("Contact Name").
					Description("Primary contact person (required)").
					Value(&shipFrom.Name).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("contact name is required")
						}
						return nil
					}),
				
				huh.NewInput().
					Title("Email").
					Description("Contact email (optional)").
					Value(&shipFrom.Email),
				
				huh.NewInput().
					Title("Phone Number").
					Description("Contact phone number (required)").
					Value(&shipFrom.PhoneNumber).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("phone number is required")
						}
						return nil
					}),
			),
		)
		
		if err := form.Run(); err != nil {
			return err
		}
		
		if shipFrom.CountryCode == "" {
			shipFrom.CountryCode = "US"
		}
		
		if i == 0 {
			shipFrom.Default = true
		} else {
			huh.NewConfirm().
				Title("Make this the default ship-from address?").
				Value(&shipFrom.Default).
				Run()
		}
		
		newConfig.Amazon.FBA.ShipFrom = append(newConfig.Amazon.FBA.ShipFrom, shipFrom)
	}
	
	return nil
}

func configureOptionalServices(newConfig *config.Cfg) error {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B82F6")).
		Bold(true).
		MarginTop(1)
	
	fmt.Printf("%s\n", headerStyle.Render("🔧 Optional Services"))
	
	var enableGroq bool
	err := huh.NewConfirm().
		Title("Enable Groq AI integration?").
		Description("Configure Groq AI for enhanced features").
		Value(&enableGroq).
		Run()
	
	if err != nil {
		return err
	}
	
	if enableGroq {
		var groqToken string
		err = huh.NewInput().
			Title("Groq API Token").
			Description("Your Groq API token for AI features").
			Value(&groqToken).
			EchoMode(huh.EchoModePassword).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("groq token is required when enabling Groq AI")
				}
				return nil
			}).
			Run()
		
		if err != nil {
			return err
		}
		
		newConfig.Groq.Token = groqToken
	}
	
	var enableSQLite bool
	err = huh.NewConfirm().
		Title("Enable SQLite database?").
		Description("Configure local SQLite database for caching and storage").
		Value(&enableSQLite).
		Run()
	
	if err != nil {
		return err
	}
	
	if enableSQLite {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		defaultDBPath := filepath.Join(home, ".halycon.db")
		
		var dbPath string
		err = huh.NewInput().
			Title("Database Path").
			Description("Path to SQLite database file").
			Value(&dbPath).
			Placeholder(defaultDBPath).
			Run()
		
		if err != nil {
			return err
		}
		
		if dbPath == "" {
			dbPath = defaultDBPath
		}
		newConfig.Sqlite.Path = dbPath
	}
	
	return nil
}
