// Package source resolves declarative mod specs into concrete downloadable
// artifacts. Providers (Modrinth, CurseForge, GitHub, ...) implement the
// Source interface and are looked up by name through the registry.
package source

import (
	"context"

	"github.com/harrisonablack/minecart/pkg/config"
)

// Artifact is a resolved, downloadable file.
type Artifact struct {
	Name     string
	URL      string
	Filename string
	Hashes   map[string]string // algorithm -> hex digest (e.g. "sha512": "...")
	Size     int64             // bytes; 0 when the provider does not report it
}

// Source resolves a declarative mod spec into a concrete artifact.
type Source interface {
	Resolve(ctx context.Context, spec config.ModSpec) (Artifact, error)
}
