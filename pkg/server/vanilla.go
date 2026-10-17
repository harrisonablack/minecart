package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/harrisonablack/minecart/pkg/artifact"
)

const (
	vanillaManifestURL = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"
	userAgent          = "minecart/0.1 (github.com/harrisonablack/minecart)"
)

// vanilla resolves a vanilla server jar from Mojang's version manifest.
//
// The zero value is usable; client and manifestURL default to a 30s-timeout
// client and the public Mojang manifest respectively. Tests may override them.
type vanilla struct {
	client      *http.Client
	manifestURL string
}

// resolve looks up version in the manifest, then fetches that version's
// metadata to find the server jar download.
func (v vanilla) resolve(ctx context.Context, version string) (artifact.Artifact, error) {
	client := v.client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	manifestURL := v.manifestURL
	if manifestURL == "" {
		manifestURL = vanillaManifestURL
	}

	var manifest versionManifest
	if err := getJSON(ctx, client, manifestURL, &manifest); err != nil {
		return artifact.Artifact{}, fmt.Errorf("vanilla: fetch manifest: %w", err)
	}

	versionURL := manifest.versionURL(version)
	if versionURL == "" {
		return artifact.Artifact{}, fmt.Errorf("vanilla: version %q not found", version)
	}

	var vj versionJSON
	if err := getJSON(ctx, client, versionURL, &vj); err != nil {
		return artifact.Artifact{}, fmt.Errorf("vanilla: fetch version %q: %w", version, err)
	}

	return artifact.Artifact{
		Name:     "vanilla",
		URL:      vj.Downloads.Server.URL,
		Filename: "server.jar",
		Hashes:   map[string]string{"sha1": vj.Downloads.Server.SHA1},
		Size:     vj.Downloads.Server.Size,
	}, nil
}

// versionManifest is the subset of Mojang's version manifest we need.
type versionManifest struct {
	Versions []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"versions"`
}

// versionURL returns the metadata URL for the given version, or "" if absent.
func (m versionManifest) versionURL(version string) string {
	for _, v := range m.Versions {
		if v.ID == version {
			return v.URL
		}
	}
	return ""
}

// versionJSON is the subset of a version's metadata we need.
type versionJSON struct {
	Downloads struct {
		Server struct {
			SHA1 string `json:"sha1"`
			Size int64  `json:"size"`
			URL  string `json:"url"`
		} `json:"server"`
	} `json:"downloads"`
}

// getJSON performs a GET request and decodes the JSON response into out.
func getJSON(ctx context.Context, client *http.Client, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
