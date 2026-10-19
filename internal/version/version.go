// Package version holds the minecart version and derived metadata.
package version

// These are variables rather than constants so the version can be injected at
// build time with -ldflags "-X github.com/harrisonablack/minecart/internal/version.Version=...".
var (
	// Version is the minecart version.
	Version = "0.1"

	// UserAgent is the HTTP User-Agent header used for all outbound requests.
	UserAgent = "minecart/" + Version + " (github.com/harrisonablack/minecart)"
)
