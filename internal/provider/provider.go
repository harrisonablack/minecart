// Package provider resolves declarative resource specs into concrete
// downloadable artifacts. Providers (Modrinth, CurseForge, GitHub, ...)
// implement the Provider interface and are looked up by name through the
// registry.
package provider

import (
	"context"

	"github.com/harrisonablack/minecart/internal/artifact"
	"github.com/harrisonablack/minecart/internal/config"
)

// Provider resolves a declarative resource spec into a concrete artifact.
// mcVersion is the server's concrete Minecraft version, used only for "latest"
// resolution; pinned versions ignore it.
type Provider interface {
	Resolve(ctx context.Context, spec config.ResourceSpec, mcVersion string) (artifact.Artifact, error)
}
