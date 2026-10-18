package source

import (
	"context"
	"fmt"

	"github.com/harrisonablack/minecart/internal/artifact"
	"github.com/harrisonablack/minecart/internal/config"
)

// registry maps a source name (from config) to its resolver.
var registry = map[string]Source{
	"modrinth": Modrinth{},
	// "curseforge": CurseForge{}, // future
	// "github":     GitHub{},     // future
	// "url":        URL{},        // future
}

// Resolve resolves a mod spec through the source named by spec.Source.
func Resolve(ctx context.Context, spec config.ModSpec) (artifact.Artifact, error) {
	s, ok := registry[spec.Source]
	if !ok {
		return artifact.Artifact{}, fmt.Errorf("unknown source %q", spec.Source)
	}
	return s.Resolve(ctx, spec)
}
