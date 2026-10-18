// Package source resolves declarative mod specs into concrete downloadable
// artifacts. Providers (Modrinth, CurseForge, GitHub, ...) implement the
// Source interface and are looked up by name through the registry.
package source

import (
	"context"

	"github.com/harrisonablack/minecart/internal/artifact"
	"github.com/harrisonablack/minecart/internal/config"
)

// Source resolves a declarative mod spec into a concrete artifact.
type Source interface {
	Resolve(ctx context.Context, spec config.ModSpec) (artifact.Artifact, error)
}
