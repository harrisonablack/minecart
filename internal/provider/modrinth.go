package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/harrisonablack/minecart/internal/artifact"
	"github.com/harrisonablack/minecart/internal/config"
	"github.com/harrisonablack/minecart/internal/version"
)

const modrinthBaseURL = "https://api.modrinth.com/v2"

// paperLoader is the Modrinth loader for Paper plugins. Only versions that
// declare this loader are compatible with a Paper server.
const paperLoader = "paper"

// Modrinth resolves resources from the Modrinth API.
//
// The zero value is usable; client and baseURL default to a 30s-timeout
// client and the public Modrinth API respectively. Tests may override them.
type Modrinth struct {
	client  *http.Client
	baseURL string
}

// Resolve looks up spec.Name (a project slug) on Modrinth and returns the
// version's primary file as an Artifact. A pinned spec.Version fetches that
// exact version; an empty or "latest" version resolves the newest version
// compatible with mcVersion and the spec.Channel threshold.
func (m Modrinth) Resolve(ctx context.Context, spec config.ResourceSpec, mcVersion string) (artifact.Artifact, error) {
	client := m.client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	baseURL := m.baseURL
	if baseURL == "" {
		baseURL = modrinthBaseURL
	}

	if spec.Version != "" && spec.Version != "latest" {
		return m.resolvePinned(ctx, client, baseURL, spec)
	}
	return m.resolveLatest(ctx, client, baseURL, spec, mcVersion)
}

// resolvePinned fetches the exact version named by spec.Version.
func (m Modrinth) resolvePinned(ctx context.Context, client *http.Client, baseURL string, spec config.ResourceSpec) (artifact.Artifact, error) {
	url := fmt.Sprintf("%s/project/%s/version/%s", baseURL, spec.Name, spec.Version)

	var v modrinthVersion
	if err := getJSON(ctx, client, url, &v); err != nil {
		return artifact.Artifact{}, fmt.Errorf("modrinth: %w", err)
	}

	file, ok := v.primaryFile()
	if !ok {
		return artifact.Artifact{}, fmt.Errorf("modrinth: no files for %s@%s", spec.Name, spec.Version)
	}

	versionNumber := v.VersionNumber
	if versionNumber == "" {
		versionNumber = spec.Version
	}

	return artifact.Artifact{
		Name:     spec.Name,
		Version:  versionNumber,
		URL:      file.URL,
		Filename: file.Filename,
		Hashes:   file.Hashes,
		Size:     file.Size,
	}, nil
}

// resolveLatest lists versions and picks the newest compatible with mcVersion
// and the spec.Channel threshold.
func (m Modrinth) resolveLatest(ctx context.Context, client *http.Client, baseURL string, spec config.ResourceSpec, mcVersion string) (artifact.Artifact, error) {
	url := fmt.Sprintf("%s/project/%s/version", baseURL, spec.Name)

	var versions []modrinthVersion
	if err := getJSON(ctx, client, url, &versions); err != nil {
		return artifact.Artifact{}, fmt.Errorf("modrinth: %w", err)
	}

	v, ok := latestCompatible(versions, mcVersion, spec.Channel, paperLoader)
	if !ok {
		if spec.Channel == "" || spec.Channel == "stable" {
			return artifact.Artifact{}, fmt.Errorf("modrinth: no stable release compatible with version %q", mcVersion)
		}
		return artifact.Artifact{}, fmt.Errorf("modrinth: no %s release compatible with version %q", spec.Channel, mcVersion)
	}

	file, ok := v.primaryFile()
	if !ok {
		return artifact.Artifact{}, fmt.Errorf("modrinth: no files for %s@%s", spec.Name, v.VersionNumber)
	}

	return artifact.Artifact{
		Name:     spec.Name,
		Version:  v.VersionNumber,
		URL:      file.URL,
		Filename: file.Filename,
		Hashes:   file.Hashes,
		Size:     file.Size,
	}, nil
}

// modrinthVersion is the subset of the Modrinth version object we need.
type modrinthVersion struct {
	ID            string         `json:"id"`
	VersionNumber string         `json:"version_number"`
	VersionType   string         `json:"version_type"` // release | beta | alpha
	GameVersions  []string       `json:"game_versions"`
	Loaders       []string       `json:"loaders"`
	DatePublished string         `json:"date_published"`
	Files         []modrinthFile `json:"files"`
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

// channelAllows reports whether versionType satisfies the channel threshold.
// An empty channel is treated as stable.
func channelAllows(versionType, channel string) bool {
	switch channel {
	case "", "stable":
		return versionType == "release"
	case "beta":
		return versionType == "release" || versionType == "beta"
	case "alpha":
		return versionType == "release" || versionType == "beta" || versionType == "alpha"
	default:
		return false
	}
}

// latestCompatible returns the newest version whose game_versions includes
// mcVersion, whose loaders includes loader, and whose version_type satisfies
// the channel threshold. ok is false when no version matches.
func latestCompatible(versions []modrinthVersion, mcVersion, channel, loader string) (modrinthVersion, bool) {
	var candidates []modrinthVersion
	for _, v := range versions {
		if !contains(v.GameVersions, mcVersion) {
			continue
		}
		if !contains(v.Loaders, loader) {
			continue
		}
		if !channelAllows(v.VersionType, channel) {
			continue
		}
		candidates = append(candidates, v)
	}
	if len(candidates) == 0 {
		return modrinthVersion{}, false
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].DatePublished > candidates[j].DatePublished
	})
	return candidates[0], true
}

// contains reports whether s contains target.
func contains(s []string, target string) bool {
	for _, v := range s {
		if v == target {
			return true
		}
	}
	return false
}

// getJSON performs a GET request and decodes the JSON response into out.
func getJSON(ctx context.Context, client *http.Client, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", version.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
