package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Load reads configuration from the file at path, then overlays environment
// variables (MINECART_*), which take precedence over file values.
func Load(path string) (*Config, error) {
	k := koanf.New(".")

	if err := loadFile(k, path); err != nil {
		return nil, err
	}

	if err := k.Load(env.Provider("MINECART_", ".", envTransform), nil); err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// loadFile loads the file at path, choosing a parser from its extension.
func loadFile(k *koanf.Koanf, path string) error {
	var parser koanf.Parser
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yml", ".yaml":
		parser = yaml.Parser()
	case ".json":
		parser = json.Parser()
	default:
		return fmt.Errorf("unsupported config format %q (want .yml, .yaml, or .json)", filepath.Ext(path))
	}

	if err := k.Load(file.Provider(path), parser); err != nil {
		return fmt.Errorf("load config file %q: %w", path, err)
	}
	return nil
}

// envTransform maps MINECART_SERVER_MEMORY to server.memory.
func envTransform(s string) string {
	s = strings.TrimPrefix(s, "MINECART_")
	return strings.ReplaceAll(strings.ToLower(s), "_", ".")
}
