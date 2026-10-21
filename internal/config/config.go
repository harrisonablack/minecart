package config

import (
	"errors"
	"fmt"
)

// Config is the top-level minecart configuration.
type Config struct {
	Directory string          `koanf:"directory"`
	Server    ServerConfig    `koanf:"server"`
	Resources ResourcesConfig `koanf:"resources"`
	Config    FileConfig      `koanf:"config"`
}

// ResourcesConfig declares resources to install alongside the server. Plugins
// is the first resource type; future types (mods, datapacks, ...) slot in as
// siblings.
type ResourcesConfig struct {
	// Sync removes manifest-tracked resources that are no longer declared.
	Sync    bool           `koanf:"sync"`
	Plugins []ResourceSpec `koanf:"plugins"`
}

// FileConfig declares values to merge into files relative to Directory.
// Filenames and setting keys are literal, including any dots.
type FileConfig struct {
	Files map[string]map[string]any `koanf:"files"`
}

// ServerConfig describes the Minecraft server to install and run.
type ServerConfig struct {
	Type    string `koanf:"type"`
	Version string `koanf:"version"`
	Eula    bool   `koanf:"eula"`
	Memory  string `koanf:"memory"`
	// JvmFlags are raw JVM flags placed before -jar. Like Memory, they only
	// apply to java entries.
	JvmFlags []string `koanf:"jvm_flags"`
	Args     []string `koanf:"args"`
}

// ResourceSpec is a declarative description of a single resource to install.
type ResourceSpec struct {
	Name     string `koanf:"name"`
	Provider string `koanf:"provider"`
	Version  string `koanf:"version"` // empty or "latest" -> latest compatible
	Channel  string `koanf:"channel"` // stable | beta | alpha; default stable
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
	for i, p := range c.Resources.Plugins {
		if p.Name == "" {
			return fmt.Errorf("resources.plugins[%d].name is required", i)
		}
		if p.Provider == "" {
			return fmt.Errorf("resources.plugins[%d].provider is required", i)
		}
		switch p.Channel {
		case "", "stable", "beta", "alpha":
		default:
			return fmt.Errorf("resources.plugins[%d].channel must be one of stable, beta, alpha (got %q)", i, p.Channel)
		}
	}
	return nil
}
