package config

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"

	// for some godforsaken reason linter cannot find yaml.v3 without this import
	// even though it is clearly defined in go.mod
	yaml "gopkg.in/yaml.v3"
)

// SnapshotToDisk writes the current configuration to disk
func SnapshotToDisk() error {
	yamlData, err := yaml.Marshal(Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to yaml: %w", err)
	}

	if err := os.WriteFile(Config.Path, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	log.Trace().Str("path", Config.Path).Msg("snapshotted config to disk")
	return nil
}
