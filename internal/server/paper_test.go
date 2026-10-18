package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
	art, err := p.resolve(context.Background(), "1.21.4")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if art.Name != "paper" {
		t.Errorf("Name = %q, want %q", art.Name, "paper")
	}
	if art.URL != "https://fill-data.papermc.io/v1/objects/abc123/paper-1.21.4-232.jar" {
		t.Errorf("URL = %q", art.URL)
	}
	if art.Filename != "server.jar" {
		t.Errorf("Filename = %q, want %q", art.Filename, "server.jar")
	}
	if art.Size != 51437498 {
		t.Errorf("Size = %d, want 51437498", art.Size)
	}
	if art.Hashes["sha256"] != "abc123" {
		t.Errorf("Hashes[sha256] = %q, want %q", art.Hashes["sha256"], "abc123")
	}
}

func TestResolvePaperPicksStable(t *testing.T) {
	srv := newPaperServer(t, `[
		{"id": 233, "channel": "EXPERIMENTAL", "downloads": {"server:default": {"name": "exp.jar", "checksums": {"sha256": "exp"}, "size": 1, "url": "https://example.com/exp.jar"}}},
		{"id": 232, "channel": "STABLE", "downloads": {"server:default": {"name": "stable.jar", "checksums": {"sha256": "stable"}, "size": 2, "url": "https://example.com/stable.jar"}}}
	]`)

	p := paper{baseURL: srv.URL}
	art, err := p.resolve(context.Background(), "1.21.4")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if art.URL != "https://example.com/stable.jar" {
		t.Errorf("URL = %q, want stable build URL", art.URL)
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
