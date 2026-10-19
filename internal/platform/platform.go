// Package platform resolves the Minecraft server jar for a given server config.
// Unlike mods (provider), server types are a fixed set resolved by a switch.
package platform

import (
	"context"
	"fmt"
	"net/url"
	"path"

	"github.com/harrisonablack/minecart/internal/artifact"
	"github.com/harrisonablack/minecart/internal/config"
	"github.com/harrisonablack/minecart/internal/manifest"
)

// Resolved is a resolved server: what to download, how to launch it, and a
// stable identity used to detect updates.
type Resolved struct {
	Artifact artifact.Artifact
	Entry    manifest.Entry
	// Identity is an opaque, platform-defined id that changes whenever the
	// resolved server changes (e.g. a new build).
	Identity string
}

// Resolve resolves the server for the given server config.
func Resolve(ctx context.Context, cfg config.ServerConfig) (Resolved, error) {
	switch cfg.Type {
	case "vanilla":
		return vanilla{}.resolve(ctx, cfg.Version)
	case "paper":
		return paper{}.resolve(ctx, cfg.Version)
	default:
		return Resolved{}, fmt.Errorf("unknown server type %q", cfg.Type)
	}
}

// filenameFromURL returns the last path segment of rawURL, ignoring any query
// or fragment.
func filenameFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse url %q: %w", rawURL, err)
	}
	name := path.Base(u.Path)
	if name == "." || name == "/" || name == "" {
		return "", fmt.Errorf("no filename in url %q", rawURL)
	}
	return name, nil
}
