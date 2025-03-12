package cmd

import (
	"fmt"
	"os"

	"github.com/caner-cetin/halycon/internal/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "halycon",
	Short: "utility tools for amazon seller API",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var (
	verbosity int
)

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.halycon.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.AddCommand(getlookupAsinFromUpcCmd())
	rootCmd.AddCommand(getLookupSkuFromAsinCmd())
	rootCmd.AddCommand(getShipmentCmd())
	rootCmd.AddCommand(getDefinitionsCmd())
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	rootCmd.PersistentFlags().CountVarP(&verbosity, "verbose", "v", "verbose output (-v: info, -vv: debug, -vvv: trace)")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".halycon" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".halycon")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	viper.SetDefault(config.AMAZON_AUTH_ENDPOINT.Key, config.AMAZON_AUTH_ENDPOINT.Default)
	viper.SetDefault(config.AMAZON_AUTH_HOST.Key, config.AMAZON_AUTH_HOST.Default)
	viper.SetDefault(config.AMAZON_MARKETPLACE_ID.Key, []string{config.AMAZON_MARKETPLACE_ID.Default})
	viper.SetDefault(config.AMAZON_FBA_SHIP_FROM_COUNTRY_CODE.Key, config.AMAZON_FBA_SHIP_FROM_COUNTRY_CODE.Default)

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	switch verbosity {
	case 1:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case 2:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case 3:
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}
}
