package platform

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/harrisonablack/minecart/internal/artifact"
	"github.com/harrisonablack/minecart/internal/manifest"
)

const paperBaseURL = "https://fill.papermc.io/v3"

// paper resolves a Paper server jar from the PaperMC downloads API.
//
// The zero value is usable; client and baseURL default to a 30s-timeout
// client and the public PaperMC API respectively. Tests may override them.
type paper struct {
	client  *http.Client
	baseURL string
}

// resolve fetches the builds for version and returns the latest stable build's
// server jar. LatestVersion resolves to the newest version with a stable build.
func (p paper) resolve(ctx context.Context, version string) (Resolved, error) {
	client := p.client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	baseURL := p.baseURL
	if baseURL == "" {
		baseURL = paperBaseURL
	}

	var (
		build paperBuild
		err   error
	)
	if version == LatestVersion {
		version, build, err = latestStableVersion(ctx, client, baseURL)
		if err != nil {
			return Resolved{}, fmt.Errorf("paper: %w", err)
		}
	} else {
		var ok bool
		build, ok, err = stableBuild(ctx, client, baseURL, version)
		if err != nil {
			return Resolved{}, fmt.Errorf("paper: %w", err)
		}
		if !ok {
			return Resolved{}, fmt.Errorf("paper: no stable build for version %q", version)
		}
	}

	dl, ok := build.Downloads["server:default"]
	if !ok {
		return Resolved{}, fmt.Errorf("paper: no server:default download for build %d", build.ID)
	}

	filename, err := filenameFromURL(dl.URL)
	if err != nil {
		return Resolved{}, fmt.Errorf("paper: %w", err)
	}

	return Resolved{
		Artifact: artifact.Artifact{
			Name:     "paper",
			URL:      dl.URL,
			Filename: filename,
			Hashes:   map[string]string{"sha256": dl.Checksums.SHA256},
			Size:     dl.Size,
		},
		Entry:            manifest.Entry{Type: manifest.EntryJava, Point: filename},
		Identity:         fmt.Sprintf("paper-%s-%d", version, build.ID),
		MinecraftVersion: version,
	}, nil
}

// latestStableVersion returns the newest version that has a stable build,
// along with that build. The newest versions may only have experimental builds
// (release candidates, or a release Paper hasn't marked stable yet), so older
// versions are tried in turn.
func latestStableVersion(ctx context.Context, client *http.Client, baseURL string) (string, paperBuild, error) {
	var vs paperVersions
	if err := getJSON(ctx, client, baseURL+"/projects/paper/versions", &vs); err != nil {
		return "", paperBuild{}, fmt.Errorf("fetch versions: %w", err)
	}
	if len(vs.Versions) == 0 {
		return "", paperBuild{}, errors.New("no versions available")
	}

	for _, v := range vs.Versions {
		build, ok, err := stableBuild(ctx, client, baseURL, v.Version.ID)
		if err != nil {
			return "", paperBuild{}, err
		}
		if ok {
			return v.Version.ID, build, nil
		}
	}
	return "", paperBuild{}, errors.New("no version has a stable build")
}

// stableBuild fetches the builds for version and returns the latest stable
// one. ok is false if the version has no stable build.
func stableBuild(ctx context.Context, client *http.Client, baseURL, version string) (build paperBuild, ok bool, err error) {
	url := fmt.Sprintf("%s/projects/paper/versions/%s/builds", baseURL, version)

	var builds []paperBuild
	if err := getJSON(ctx, client, url, &builds); err != nil {
		return paperBuild{}, false, fmt.Errorf("fetch builds for version %q: %w", version, err)
	}

	build, ok = latestStable(builds)
	return build, ok, nil
}

// paperVersions is the subset of the PaperMC versions list we need. Versions
// are listed newest-first.
type paperVersions struct {
	Versions []struct {
		Version struct {
			ID string `json:"id"`
		} `json:"version"`
	} `json:"versions"`
}

// paperBuild is the subset of the PaperMC build object we need.
type paperBuild struct {
	ID        int                      `json:"id"`
	Channel   string                   `json:"channel"`
	Downloads map[string]paperDownload `json:"downloads"`
}

// paperDownload is the subset of the PaperMC download object we need.
type paperDownload struct {
	Name      string `json:"name"`
	Checksums struct {
		SHA256 string `json:"sha256"`
	} `json:"checksums"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

// latestStable returns the first STABLE build (the list is newest-first).
func latestStable(builds []paperBuild) (paperBuild, bool) {
	for _, b := range builds {
		if b.Channel == "STABLE" {
			return b, true
		}
	}
	return paperBuild{}, false
}
