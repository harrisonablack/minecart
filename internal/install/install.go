// Package install resolves, downloads, verifies, and prepares a Minecraft
// server for running.
package install

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/harrisonablack/minecart/internal/config"
	"github.com/harrisonablack/minecart/internal/platform"
)

// Run resolves the server jar, downloads and verifies it, and writes eula.txt.
func Run(ctx context.Context, logger *slog.Logger, cfg config.Config) error {
	art, err := platform.Resolve(ctx, cfg.Server)
	if err != nil {
		return fmt.Errorf("resolve server: %w", err)
	}
	logger.Info("resolved server", "artifact", art.Name, "version", cfg.Server.Version)

	if err := Download(ctx, logger, art, cfg.Directory); err != nil {
		return err
	}

	if err := writeEULA(cfg.Directory); err != nil {
		return err
	}
	logger.Info("wrote eula.txt", "directory", cfg.Directory)

	return nil
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
