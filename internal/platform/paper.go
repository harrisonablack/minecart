package platform

import (
	"context"
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
// server jar.
func (p paper) resolve(ctx context.Context, version string) (Resolved, error) {
	client := p.client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	baseURL := p.baseURL
	if baseURL == "" {
		baseURL = paperBaseURL
	}

	url := fmt.Sprintf("%s/projects/paper/versions/%s/builds", baseURL, version)

	var builds []paperBuild
	if err := getJSON(ctx, client, url, &builds); err != nil {
		return Resolved{}, fmt.Errorf("paper: fetch builds: %w", err)
	}

	build, ok := latestStable(builds)
	if !ok {
		return Resolved{}, fmt.Errorf("paper: no stable build for version %q", version)
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
		Entry:    manifest.Entry{Type: manifest.EntryJava, Point: filename},
		Identity: fmt.Sprintf("paper-%s-%d", version, build.ID),
	}, nil
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
