// Package server resolves the Minecraft server jar for a given server config.
// Unlike mods (pkg/source), server types are a fixed set resolved by a switch.
package server

import (
	"context"
	"fmt"

	"github.com/harrisonablack/minecart/pkg/artifact"
	"github.com/harrisonablack/minecart/pkg/config"
)

// Resolve resolves the server jar for the given server config.
func Resolve(ctx context.Context, cfg config.ServerConfig) (artifact.Artifact, error) {
	switch cfg.Type {
	case "vanilla":
		return vanilla{}.resolve(ctx, cfg.Version)
	default:
		return artifact.Artifact{}, fmt.Errorf("unknown server type %q", cfg.Type)
	}
}
