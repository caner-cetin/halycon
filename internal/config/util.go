package config

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	// for some godforsaken reason linter cannot find yaml.v3 without this import
	// even though it is clearly defined in go.mod
	yaml "gopkg.in/yaml.v3"
)

// SnapshotToDisk writes the current configuration to disk and reloads the Viper instance
func SnapshotToDisk() error {
	configPath := viper.ConfigFileUsed()
	yamlData, err := yaml.Marshal(Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to yaml: %w", err)
	}
	if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reloading viper config: %w", err)
	}
	log.Trace().Str("path", configPath).Msg("snapshotted config to disk")
	return nil
}
