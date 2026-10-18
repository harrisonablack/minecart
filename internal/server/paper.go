package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/harrisonablack/minecart/internal/artifact"
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
// server jar as an Artifact.
func (p paper) resolve(ctx context.Context, version string) (artifact.Artifact, error) {
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
		return artifact.Artifact{}, fmt.Errorf("paper: fetch builds: %w", err)
	}

	build, ok := latestStable(builds)
	if !ok {
		return artifact.Artifact{}, fmt.Errorf("paper: no stable build for version %q", version)
	}

	dl, ok := build.Downloads["server:default"]
	if !ok {
		return artifact.Artifact{}, fmt.Errorf("paper: no server:default download for build %d", build.ID)
	}

	return artifact.Artifact{
		Name:     "paper",
		URL:      dl.URL,
		Filename: "server.jar",
		Hashes:   map[string]string{"sha256": dl.Checksums.SHA256},
		Size:     dl.Size,
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
