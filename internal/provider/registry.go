package provider

import (
	"context"
	"fmt"

	"github.com/harrisonablack/minecart/internal/artifact"
	"github.com/harrisonablack/minecart/internal/config"
)

// registry maps a provider name (from config) to its resolver.
var registry = map[string]Provider{
	"modrinth": Modrinth{},
	// "curseforge": CurseForge{}, // future
	// "github":     GitHub{},     // future
	// "url":        URL{},        // future
}

// Resolve resolves a resource spec through the provider named by spec.Provider.
func Resolve(ctx context.Context, spec config.ResourceSpec, mcVersion string) (artifact.Artifact, error) {
	s, ok := registry[spec.Provider]
	if !ok {
		return artifact.Artifact{}, fmt.Errorf("unknown provider %q", spec.Provider)
	}
	return s.Resolve(ctx, spec, mcVersion)
}
