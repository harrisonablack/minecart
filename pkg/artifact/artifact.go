// Package artifact defines the shared type for a resolved, downloadable file.
// Both mod resolution (pkg/source) and server jar resolution (pkg/server)
// produce an Artifact.
package artifact

// Artifact is a resolved, downloadable file.
type Artifact struct {
	Name     string
	URL      string
	Filename string
	Hashes   map[string]string // algorithm -> hex digest (e.g. "sha512": "...")
	Size     int64             // bytes; 0 when the provider does not report it
}
