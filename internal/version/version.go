// Package version holds the minecart version and derived metadata.
package version

const (
	// Version is the minecart version.
	Version = "0.1"

	// UserAgent is the HTTP User-Agent header used for all outbound requests.
	UserAgent = "minecart/" + Version + " (github.com/harrisonablack/minecart)"
)
