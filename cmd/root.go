package cmd

import (
	"os"

	"github.com/caner-cetin/halycon/internal/config"
	"github.com/fatih/color"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfg = &config.Config // i seriously dont want to write config.Config.
var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "halycon",
	Short: "utility tools for amazon seller API",
}

var versionCmd = &cobra.Command{
	Use: "version",
	Run: displayVersion,
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
	rootCmd.AddCommand(getListingsCmd())
	rootCmd.AddCommand(getCatalogCmd())
	rootCmd.AddCommand(versionCmd)

	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	rootCmd.PersistentFlags().CountVarP(&verbosity, "verbose", "v", "verbose output (-v: info, -vv: debug, -vvv: trace)")
	rootCmd.PersistentFlags()

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
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
		log.Info().Str("path", viper.ConfigFileUsed()).Msg("reading config")
	} else {
		log.Error().Err(err).Msg("failed to read config")
		return
	}

	if err := viper.Unmarshal(&config.Config); err != nil {
		log.Error().Err(err).Msg("error unmarshalling config")
		return
	}
	if err := config.SetDefaultClient(); err != nil {
		log.Error().Err(err).Msg("failed to set default client")
		return
	}
	if err := config.SetDefaultMerchant(); err != nil {
		log.Error().Err(err).Msg("failed to set default merchant")
		return
	}
	if err := config.SetDefaultShipFromAddress(); err != nil {
		log.Error().Err(err).Msg("failed to set default ship from address")
		return
	}
	config.SetOtherDefaults()
	if err := config.SnapshotToDisk(); err != nil {
		log.Error().Err(err).Msg("failed to write config to disk")
		return
	}
}
func displayVersion(cmd *cobra.Command, args []string) {
	_, err := color.New(color.Bold).Println("Halycon 0.2.0")
	if err != nil {
		// how the fuck does this even throw error
		log.Error().Err(err).Send()
	}
}
