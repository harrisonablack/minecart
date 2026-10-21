package platform

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/harrisonablack/minecart/internal/artifact"
	"github.com/harrisonablack/minecart/internal/httpx"
	"github.com/harrisonablack/minecart/internal/manifest"
)

const vanillaManifestURL = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"

// vanilla resolves a vanilla server jar from Mojang's version manifest.
//
// The zero value is usable; client and manifestURL default to a 30s-timeout
// client and the public Mojang manifest respectively. Tests may override them.
type vanilla struct {
	client      *http.Client
	manifestURL string
}

// resolve looks up version in the manifest, then fetches that version's
// metadata to find the server jar download. LatestVersion resolves to the
// manifest's latest release.
func (v vanilla) resolve(ctx context.Context, version string) (Resolved, error) {
	client := v.client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	manifestURL := v.manifestURL
	if manifestURL == "" {
		manifestURL = vanillaManifestURL
	}

	var vm versionManifest
	if err := httpx.GetJSON(ctx, client, manifestURL, &vm); err != nil {
		return Resolved{}, fmt.Errorf("vanilla: fetch manifest: %w", err)
	}

	if version == LatestVersion {
		version = vm.Latest.Release
		if version == "" {
			return Resolved{}, errors.New("vanilla: no latest release in manifest")
		}
	}

	versionURL := vm.versionURL(version)
	if versionURL == "" {
		return Resolved{}, fmt.Errorf("vanilla: version %q not found", version)
	}

	var vj versionJSON
	if err := httpx.GetJSON(ctx, client, versionURL, &vj); err != nil {
		return Resolved{}, fmt.Errorf("vanilla: fetch version %q: %w", version, err)
	}

	filename, err := filenameFromURL(vj.Downloads.Server.URL)
	if err != nil {
		return Resolved{}, fmt.Errorf("vanilla: %w", err)
	}

	return Resolved{
		Artifact: artifact.Artifact{
			Name:     "vanilla",
			URL:      vj.Downloads.Server.URL,
			Filename: filename,
			Hashes:   map[string]string{"sha1": vj.Downloads.Server.SHA1},
			Size:     vj.Downloads.Server.Size,
		},
		Entry:            manifest.Entry{Type: manifest.EntryJava, Point: filename},
		Identity:         "vanilla-" + version,
		MinecraftVersion: version,
	}, nil
}

// versionManifest is the subset of Mojang's version manifest we need.
type versionManifest struct {
	Latest struct {
		Release  string `json:"release"`
		Snapshot string `json:"snapshot"`
	} `json:"latest"`
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
