package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harrisonablack/minecart/pkg/config"
)

// writeResponse writes body to w, failing the test on error.
func writeResponse(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if _, err := w.Write([]byte(body)); err != nil {
		t.Errorf("write response: %v", err)
	}
}

// newVanillaServer serves a manifest at /manifest and a version at /version,
// with the manifest's version URL pointing back at the test server.
func newVanillaServer(t *testing.T, versionJSON string) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest":
			writeResponse(t, w, `{"versions": [{"id": "26.3", "url": "`+srv.URL+`/version"}]}`)
		case "/version":
			writeResponse(t, w, versionJSON)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestResolveVanilla(t *testing.T) {
	srv := newVanillaServer(t, `{
		"downloads": {"server": {
			"sha1": "abc123",
			"size": 62294556,
			"url": "https://piston-data.mojang.com/v1/objects/abc123/server.jar"
		}}
	}`)

	v := vanilla{manifestURL: srv.URL + "/manifest"}
	art, err := v.resolve(context.Background(), "26.3")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if art.Name != "vanilla" {
		t.Errorf("Name = %q, want %q", art.Name, "vanilla")
	}
	if art.URL != "https://piston-data.mojang.com/v1/objects/abc123/server.jar" {
		t.Errorf("URL = %q", art.URL)
	}
	if art.Filename != "server.jar" {
		t.Errorf("Filename = %q, want %q", art.Filename, "server.jar")
	}
	if art.Size != 62294556 {
		t.Errorf("Size = %d, want 62294556", art.Size)
	}
	if art.Hashes["sha1"] != "abc123" {
		t.Errorf("Hashes[sha1] = %q, want %q", art.Hashes["sha1"], "abc123")
	}
}

func TestResolveVanillaVersionNotFound(t *testing.T) {
	srv := newVanillaServer(t, `{}`)

	v := vanilla{manifestURL: srv.URL + "/manifest"}
	if _, err := v.resolve(context.Background(), "1.0.0"); err == nil {
		t.Fatal("resolve: expected error for unknown version, got nil")
	}
}

func TestResolveUnknownType(t *testing.T) {
	_, err := Resolve(context.Background(), config.ServerConfig{Type: "paper", Version: "1.21.4"})
	if err == nil {
		t.Fatal("Resolve: expected error for unknown type, got nil")
	}
}
