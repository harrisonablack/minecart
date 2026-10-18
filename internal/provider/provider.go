// Package provider resolves declarative mod specs into concrete downloadable
// artifacts. Providers (Modrinth, CurseForge, GitHub, ...) implement the
// Provider interface and are looked up by name through the registry.
package provider

import (
	"context"

	"github.com/harrisonablack/minecart/internal/artifact"
	"github.com/harrisonablack/minecart/internal/config"
)

// Provider resolves a declarative mod spec into a concrete artifact.
type Provider interface {
	Resolve(ctx context.Context, spec config.ModSpec) (artifact.Artifact, error)
}
