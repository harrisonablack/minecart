package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// defaults are the global default values, overridden by the file and then
// environment variables.
var defaults = map[string]any{
	"directory": "server",
}

// Load reads configuration from the file at path, then overlays environment
// variables (MINECART_*), which take precedence over file values.
func Load(path string) (*Config, error) {
	k := koanf.New(".")

	if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
		return nil, fmt.Errorf("load defaults: %w", err)
	}

	if err := loadFile(k, path); err != nil {
		return nil, err
	}

	if err := k.Load(env.Provider("MINECART_", ".", envTransform), nil); err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}

	// Check declaration shapes before the weakly typed application decoder can
	// coerce them (for example, a list of maps into a single map).
	if err := validateFileMaps(k.Raw()["config"]); err != nil {
		return nil, err
	}

	// Decode the nested root, not flattened key paths. Koanf's lookup delimiter
	// does not split literal filenames or setting keys in this representation.
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validateFileMaps(value any) error {
	if value == nil {
		return nil
	}
	section, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("config must be a mapping")
	}
	if section["files"] == nil {
		return nil
	}
	files, ok := section["files"].(map[string]any)
	if !ok {
		return fmt.Errorf("config.files must be a mapping")
	}
	for name, values := range files {
		if _, ok := values.(map[string]any); !ok {
			return fmt.Errorf("config.files[%q] must be a mapping", name)
		}
	}
	return nil
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
	// This variable selects the input file; it is not a configuration value.
	if s == "MINECART_CONFIG" {
		return ""
	}
	s = strings.TrimPrefix(s, "MINECART_")
	return strings.ReplaceAll(strings.ToLower(s), "_", ".")
}
