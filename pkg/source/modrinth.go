package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/harrisonablack/minecart/pkg/artifact"
	"github.com/harrisonablack/minecart/pkg/config"
	"github.com/harrisonablack/minecart/pkg/version"
)

const modrinthBaseURL = "https://api.modrinth.com/v2"

// Modrinth resolves mods from the Modrinth API.
//
// The zero value is usable; client and baseURL default to a 30s-timeout
// client and the public Modrinth API respectively. Tests may override them.
type Modrinth struct {
	client  *http.Client
	baseURL string
}

// Resolve looks up spec.Name (a project slug) and spec.Version on Modrinth
// and returns the version's primary file as an Artifact.
func (m Modrinth) Resolve(ctx context.Context, spec config.ModSpec) (artifact.Artifact, error) {
	client := m.client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	baseURL := m.baseURL
	if baseURL == "" {
		baseURL = modrinthBaseURL
	}

	url := fmt.Sprintf("%s/project/%s/version/%s", baseURL, spec.Name, spec.Version)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return artifact.Artifact{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", version.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return artifact.Artifact{}, fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return artifact.Artifact{}, fmt.Errorf("modrinth: unexpected status %d for %s@%s", resp.StatusCode, spec.Name, spec.Version)
	}

	var v modrinthVersion
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return artifact.Artifact{}, fmt.Errorf("decode response: %w", err)
	}

	file, ok := v.primaryFile()
	if !ok {
		return artifact.Artifact{}, fmt.Errorf("modrinth: no files for %s@%s", spec.Name, spec.Version)
	}

	return artifact.Artifact{
		Name:     spec.Name,
		URL:      file.URL,
		Filename: file.Filename,
		Hashes:   file.Hashes,
		Size:     file.Size,
	}, nil
}

// modrinthVersion is the subset of the Modrinth version object we need.
type modrinthVersion struct {
	Files []modrinthFile `json:"files"`
}

// modrinthFile is the subset of the Modrinth file object we need.
type modrinthFile struct {
	Hashes   map[string]string `json:"hashes"`
	URL      string            `json:"url"`
	Filename string            `json:"filename"`
	Primary  bool              `json:"primary"`
	Size     int64             `json:"size"`
}

// primaryFile returns the file flagged primary, falling back to the first
// file when none is flagged.
func (v modrinthVersion) primaryFile() (modrinthFile, bool) {
	for _, f := range v.Files {
		if f.Primary {
			return f, true
		}
	}
	if len(v.Files) > 0 {
		return v.Files[0], true
	}
	return modrinthFile{}, false
}
