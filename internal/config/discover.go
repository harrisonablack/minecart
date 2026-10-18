package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Discover returns the path to the config file to load.
//
// If MINECART_CONFIG is set, it is used directly. Otherwise, dir is searched
// for minecart.yml, minecart.yaml, or minecart.json.
func Discover(dir string) (string, error) {
	if p := os.Getenv("MINECART_CONFIG"); p != "" {
		return p, nil
	}
	for _, name := range []string{"minecart.yml", "minecart.yaml", "minecart.json"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no config file found in %s (set MINECART_CONFIG or create minecart.yml)", dir)
}

// LoadDefault discovers and loads the config file from the current directory.
func LoadDefault() (*Config, error) {
	path, err := Discover(".")
	if err != nil {
		return nil, err
	}
	return Load(path)
}
