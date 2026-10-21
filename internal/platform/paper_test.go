package platform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harrisonablack/minecart/internal/manifest"
)

// newPaperServer serves a builds list at the PaperMC builds path.
func newPaperServer(t *testing.T, buildsJSON string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/projects/paper/versions/1.21.4/builds" {
			writeResponse(t, w, buildsJSON)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newPaperVersionsServer serves versionsJSON at the PaperMC versions path, and
// each entry of builds (version -> builds list) at that version's builds path.
func newPaperVersionsServer(t *testing.T, versionsJSON string, builds map[string]string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /projects/paper/versions", func(w http.ResponseWriter, r *http.Request) {
		writeResponse(t, w, versionsJSON)
	})
	mux.HandleFunc("GET /projects/paper/versions/{version}/builds", func(w http.ResponseWriter, r *http.Request) {
		body, ok := builds[r.PathValue("version")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeResponse(t, w, body)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

const (
	paperStableBuildsJSON = `[
		{"id": 206, "channel": "STABLE", "downloads": {"server:default": {"name": "paper-26.3-206.jar", "checksums": {"sha256": "abc123"}, "size": 1, "url": "https://example.com/paper-26.3-206.jar"}}}
	]`
	paperAlphaBuildsJSON = `[
		{"id": 1, "channel": "ALPHA", "downloads": {"server:default": {"name": "alpha.jar", "checksums": {"sha256": "alpha"}, "size": 1, "url": "https://example.com/alpha.jar"}}}
	]`
)

func TestResolvePaperLatest(t *testing.T) {
	srv := newPaperVersionsServer(t,
		`{"versions": [{"version": {"id": "26.3"}}, {"version": {"id": "26.2"}}]}`,
		map[string]string{"26.3": paperStableBuildsJSON},
	)

	p := paper{baseURL: srv.URL}
	res, err := p.resolve(context.Background(), LatestVersion)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if res.Artifact.URL != "https://example.com/paper-26.3-206.jar" {
		t.Errorf("URL = %q", res.Artifact.URL)
	}
	if res.Identity != "paper-26.3-206" {
		t.Errorf("Identity = %q, want %q", res.Identity, "paper-26.3-206")
	}
	if res.MinecraftVersion != "26.3" {
		t.Errorf("MinecraftVersion = %q, want %q", res.MinecraftVersion, "26.3")
	}
}

func TestResolvePaperLatestSkipsUnstableVersions(t *testing.T) {
	srv := newPaperVersionsServer(t,
		`{"versions": [{"version": {"id": "26.4"}}, {"version": {"id": "26.4-rc-1"}}, {"version": {"id": "26.3"}}]}`,
		map[string]string{
			"26.4":      paperAlphaBuildsJSON,
			"26.4-rc-1": paperAlphaBuildsJSON,
			"26.3":      paperStableBuildsJSON,
		},
	)

	p := paper{baseURL: srv.URL}
	res, err := p.resolve(context.Background(), LatestVersion)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if res.Identity != "paper-26.3-206" {
		t.Errorf("Identity = %q, want %q", res.Identity, "paper-26.3-206")
	}
}

func TestResolvePaperLatestNoVersions(t *testing.T) {
	srv := newPaperVersionsServer(t, `{"versions": []}`, nil)

	p := paper{baseURL: srv.URL}
	if _, err := p.resolve(context.Background(), LatestVersion); err == nil {
		t.Fatal("resolve: expected error for no versions, got nil")
	}
}

func TestResolvePaperLatestNoStable(t *testing.T) {
	srv := newPaperVersionsServer(t,
		`{"versions": [{"version": {"id": "26.4"}}]}`,
		map[string]string{"26.4": paperAlphaBuildsJSON},
	)

	p := paper{baseURL: srv.URL}
	if _, err := p.resolve(context.Background(), LatestVersion); err == nil {
		t.Fatal("resolve: expected error for no stable build, got nil")
	}
}

func TestResolvePaperLatestBuildsError(t *testing.T) {
	srv := newPaperVersionsServer(t,
		`{"versions": [{"version": {"id": "26.4"}}, {"version": {"id": "26.3"}}]}`,
		map[string]string{"26.3": paperStableBuildsJSON},
	)

	p := paper{baseURL: srv.URL}
	if _, err := p.resolve(context.Background(), LatestVersion); err == nil {
		t.Fatal("resolve: expected error when fetching builds fails, got nil")
	}
}

func TestResolvePaper(t *testing.T) {
	srv := newPaperServer(t, `[{
		"id": 232,
		"channel": "STABLE",
		"downloads": {
			"server:default": {
				"name": "paper-1.21.4-232.jar",
				"checksums": {"sha256": "abc123"},
				"size": 51437498,
				"url": "https://fill-data.papermc.io/v1/objects/abc123/paper-1.21.4-232.jar"
			}
		}
	}]`)

	p := paper{baseURL: srv.URL}
	res, err := p.resolve(context.Background(), "1.21.4")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	art := res.Artifact

	if art.Name != "paper" {
		t.Errorf("Name = %q, want %q", art.Name, "paper")
	}
	if art.URL != "https://fill-data.papermc.io/v1/objects/abc123/paper-1.21.4-232.jar" {
		t.Errorf("URL = %q", art.URL)
	}
	if art.Filename != "paper-1.21.4-232.jar" {
		t.Errorf("Filename = %q, want %q", art.Filename, "paper-1.21.4-232.jar")
	}
	if art.Size != 51437498 {
		t.Errorf("Size = %d, want 51437498", art.Size)
	}
	if art.Hashes["sha256"] != "abc123" {
		t.Errorf("Hashes[sha256] = %q, want %q", art.Hashes["sha256"], "abc123")
	}
	wantEntry := manifest.Entry{Type: "java", Point: "paper-1.21.4-232.jar"}
	if res.Entry != wantEntry {
		t.Errorf("Entry = %+v, want %+v", res.Entry, wantEntry)
	}
	if res.Identity != "paper-1.21.4-232" {
		t.Errorf("Identity = %q, want %q", res.Identity, "paper-1.21.4-232")
	}
	if res.MinecraftVersion != "1.21.4" {
		t.Errorf("MinecraftVersion = %q, want %q", res.MinecraftVersion, "1.21.4")
	}
}

func TestResolvePaperPicksStable(t *testing.T) {
	srv := newPaperServer(t, `[
		{"id": 233, "channel": "EXPERIMENTAL", "downloads": {"server:default": {"name": "exp.jar", "checksums": {"sha256": "exp"}, "size": 1, "url": "https://example.com/exp.jar"}}},
		{"id": 232, "channel": "STABLE", "downloads": {"server:default": {"name": "stable.jar", "checksums": {"sha256": "stable"}, "size": 2, "url": "https://example.com/stable.jar"}}}
	]`)

	p := paper{baseURL: srv.URL}
	res, err := p.resolve(context.Background(), "1.21.4")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if res.Artifact.URL != "https://example.com/stable.jar" {
		t.Errorf("URL = %q, want stable build URL", res.Artifact.URL)
	}
	if res.Identity != "paper-1.21.4-232" {
		t.Errorf("Identity = %q, want %q", res.Identity, "paper-1.21.4-232")
	}
}

func TestResolvePaperNoStable(t *testing.T) {
	srv := newPaperServer(t, `[
		{"id": 233, "channel": "EXPERIMENTAL", "downloads": {"server:default": {"name": "exp.jar", "checksums": {"sha256": "exp"}, "size": 1, "url": "https://example.com/exp.jar"}}}
	]`)

	p := paper{baseURL: srv.URL}
	if _, err := p.resolve(context.Background(), "1.21.4"); err == nil {
		t.Fatal("resolve: expected error for no stable build, got nil")
	}
}

func TestResolvePaperNoServerDownload(t *testing.T) {
	srv := newPaperServer(t, `[
		{"id": 232, "channel": "STABLE", "downloads": {}}
	]`)

	p := paper{baseURL: srv.URL}
	if _, err := p.resolve(context.Background(), "1.21.4"); err == nil {
		t.Fatal("resolve: expected error for missing server:default, got nil")
	}
}
