// Package install resolves, downloads, verifies, and prepares a Minecraft
// server for running.
package install

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/harrisonablack/minecart/internal/config"
	"github.com/harrisonablack/minecart/internal/manifest"
	"github.com/harrisonablack/minecart/internal/platform"
)

// Run resolves the server, installs it when the manifest shows it is missing
// or out of date, and writes eula.txt.
func Run(ctx context.Context, logger *slog.Logger, cfg config.Config) error {
	resolved, err := platform.Resolve(ctx, cfg.Server)
	if err != nil {
		return fmt.Errorf("resolve server: %w", err)
	}
	logger.Info("resolved server", "artifact", resolved.Artifact.Name, "version", cfg.Server.Version,
		"identity", resolved.Identity)

	if err := installServer(ctx, logger, cfg.Server.Type, resolved, cfg.Directory); err != nil {
		return err
	}

	if err := writeEULA(cfg.Directory); err != nil {
		return err
	}
	logger.Info("wrote eula.txt", "directory", cfg.Directory)

	return nil
}

// installServer downloads the resolved server into dir unless the manifest
// already records the same identity and the entry file is intact. On change it
// rewrites the manifest and then removes the previous entry point.
func installServer(ctx context.Context, logger *slog.Logger, serverType string, resolved platform.Resolved, dir string) error {
	path := manifest.Path(dir)
	existing, err := manifest.Read(path)
	if err != nil {
		return err
	}

	dest := filepath.Join(dir, resolved.Artifact.Filename)
	if existing.Identity == resolved.Identity && fileMatches(resolved, dest) {
		logger.Info("already current", "identity", resolved.Identity, "file", dest)
		return nil
	}

	if existing.Identity != "" && existing.Identity != resolved.Identity {
		logger.Info("updating server", "from", existing.Identity, "to", resolved.Identity)
	}

	if err := Download(ctx, logger, resolved.Artifact, dir); err != nil {
		return err
	}

	m := manifest.Manifest{
		Type:        serverType,
		Identity:    resolved.Identity,
		Entry:       resolved.Entry,
		Hashes:      resolved.Artifact.Hashes,
		InstalledAt: time.Now().UTC(),
	}
	if err := manifest.Write(path, m); err != nil {
		return err
	}
	logger.Info("wrote manifest", "file", path, "identity", m.Identity)

	// Only remove the previous entry once the manifest no longer references
	// it, so a failed write leaves the old install launchable.
	if old := existing.Entry.Point; old != "" && old != resolved.Entry.Point {
		removeStale(logger, dir, old)
	}

	return nil
}

// fileMatches reports whether the resolved artifact is already present at
// dest. It verifies the hash when one is available, otherwise it only checks
// that the file exists.
func fileMatches(resolved platform.Resolved, dest string) bool {
	if algo, _ := strongestHash(resolved.Artifact.Hashes); algo != "" {
		return matches(resolved.Artifact, dest)
	}
	_, err := os.Stat(dest)
	return err == nil
}

// removeStale deletes a previous entry point from dir. Failures are logged
// rather than returned since the new server is already installed.
func removeStale(logger *slog.Logger, dir, name string) {
	if !filepath.IsLocal(name) {
		logger.Warn("not removing stale entry outside directory", "file", name)
		return
	}
	stale := filepath.Join(dir, name)
	if err := os.Remove(stale); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logger.Warn("failed to remove stale entry", "file", stale, "error", err)
		}
		return
	}
	logger.Info("removed stale entry", "file", stale)
}

// writeEULA writes eula.txt with eula=true into dir.
func writeEULA(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	path := filepath.Join(dir, "eula.txt")
	if err := os.WriteFile(path, []byte("eula=true\n"), 0o644); err != nil {
		return fmt.Errorf("write eula.txt: %w", err)
	}
	return nil
}
