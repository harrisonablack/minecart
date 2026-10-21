package install

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/harrisonablack/minecart/internal/config"
	"github.com/harrisonablack/minecart/internal/manifest"
	"github.com/harrisonablack/minecart/internal/provider"
)

// resolveResource resolves a resource spec to an artifact. It is a variable so
// tests can substitute a fake resolver.
var resolveResource = provider.Resolve

// installResources installs declared plugins alongside the server. It runs
// after installServer, reading the manifest that installServer wrote and
// updating only its Plugins field.
func installResources(ctx context.Context, logger *slog.Logger, cfg config.Config, mcVersion string) error {
	if len(cfg.Resources.Plugins) == 0 {
		return nil
	}
	if cfg.Server.Type != "paper" {
		logger.Warn("plugins declared but server is not paper; skipping plugin install", "server_type", cfg.Server.Type)
		return nil
	}

	path := manifest.Path(cfg.Directory)
	m, err := manifest.Read(path)
	if err != nil {
		return err
	}

	pluginsDir := filepath.Join(cfg.Directory, "plugins")

	// Index existing plugins by (name, provider) so declared plugins can be
	// matched and updated in place.
	existing := make(map[string]int, len(m.Plugins))
	for i, p := range m.Plugins {
		existing[pluginKey(p.Name, p.Provider)] = i
	}

	declared := make(map[string]bool, len(cfg.Resources.Plugins))

	for _, spec := range cfg.Resources.Plugins {
		art, err := resolveResource(ctx, spec, mcVersion)
		if err != nil {
			return fmt.Errorf("resolve plugin %q: %w", spec.Name, err)
		}

		key := pluginKey(spec.Name, spec.Provider)
		declared[key] = true

		if idx, ok := existing[key]; ok {
			entry := m.Plugins[idx]
			if entry.Version == art.Version && fileMatches(art, filepath.Join(pluginsDir, entry.Filename)) {
				logger.Info("plugin already current", "plugin", spec.Name, "version", art.Version)
				continue
			}
		}

		if err := Download(ctx, logger, art, pluginsDir); err != nil {
			return fmt.Errorf("download plugin %q: %w", spec.Name, err)
		}

		entry := manifest.Plugin{
			Name:     spec.Name,
			Provider: spec.Provider,
			Version:  art.Version,
			Filename: art.Filename,
			Hashes:   art.Hashes,
		}
		if idx, ok := existing[key]; ok {
			m.Plugins[idx] = entry
		} else {
			m.Plugins = append(m.Plugins, entry)
			existing[key] = len(m.Plugins) - 1
		}
	}

	if cfg.Resources.Sync {
		kept := m.Plugins[:0]
		for _, p := range m.Plugins {
			if declared[pluginKey(p.Name, p.Provider)] {
				kept = append(kept, p)
				continue
			}
			removeStale(logger, pluginsDir, p.Filename)
		}
		m.Plugins = kept
	}

	if err := manifest.Write(path, m); err != nil {
		return err
	}

	return nil
}

// pluginKey returns the identity key for a plugin: (name, provider).
func pluginKey(name, provider string) string {
	return name + "\x00" + provider
}
