package config

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Apply merges declared values into configuration files before a server launch.
// Each file is replaced atomically. If a later file fails, earlier successful
// writes remain in place and the caller must abort the launch.
func Apply(logger *slog.Logger, cfg Config) error {
	for _, name := range slices.Sorted(maps.Keys(cfg.Config.Files)) {
		if !filepath.IsLocal(name) {
			return fmt.Errorf("config.files[%q]: destination must be relative to the server directory", name)
		}
		values := cfg.Config.Files[name]
		if values == nil {
			return fmt.Errorf("config.files[%q] must be a mapping", name)
		}
		path := filepath.Join(cfg.Directory, name)
		var (
			data []byte
			err  error
		)
		switch strings.ToLower(filepath.Ext(name)) {
		case ".properties":
			data, err = mergeProperties(path, values)
		default:
			err = fmt.Errorf("unsupported configuration format %q (want .properties)", filepath.Ext(name))
		}
		if err != nil {
			return fmt.Errorf("apply config %q: %w", path, err)
		}
		if err := writeConfig(path, data); err != nil {
			return fmt.Errorf("apply config %q: %w", path, err)
		}
		logger.Info("applied configuration", "file", path)
	}
	return nil
}

// writeConfig stages the complete result alongside the destination so rename
// replaces it atomically on the same filesystem. Existing permissions survive.
func writeConfig(path string, data []byte) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat destination: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	f, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	defer func() { _ = f.Close() }()

	if err := f.Chmod(mode); err != nil {
		return fmt.Errorf("set permissions: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("replace destination: %w", err)
	}
	return nil
}
