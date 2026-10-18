package config

import (
	"errors"
	"fmt"
)

// Config is the top-level minecart configuration.
type Config struct {
	Directory string       `koanf:"directory"`
	Server    ServerConfig `koanf:"server"`
	Mods      []ModSpec    `koanf:"mods"`
}

// ServerConfig describes the Minecraft server to install and run.
type ServerConfig struct {
	Type    string   `koanf:"type"`
	Version string   `koanf:"version"`
	Eula    bool     `koanf:"eula"`
	Java    string   `koanf:"java"`
	Memory  string   `koanf:"memory"`
	Args    []string `koanf:"args"`
}

// ModSpec is a declarative description of a single mod to install.
type ModSpec struct {
	Name     string `koanf:"name"`
	Provider string `koanf:"provider"`
	Version  string `koanf:"version"`
}

// Validate reports whether the configuration is complete enough to act on.
func (c *Config) Validate() error {
	if c.Server.Type == "" {
		return errors.New("server.type is required")
	}
	if c.Server.Version == "" {
		return errors.New("server.version is required")
	}
	if !c.Server.Eula {
		return errors.New("server.eula must be true (accept the Minecraft EULA)")
	}
	for i, m := range c.Mods {
		if m.Name == "" {
			return fmt.Errorf("mods[%d].name is required", i)
		}
		if m.Provider == "" {
			return fmt.Errorf("mods[%d].provider is required", i)
		}
	}
	return nil
}
