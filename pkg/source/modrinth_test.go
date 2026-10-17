package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harrisonablack/minecart/pkg/config"
)

func newModrinthServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// writeResponse writes body to w, failing the test on error.
func writeResponse(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if _, err := w.Write([]byte(body)); err != nil {
		t.Errorf("write response: %v", err)
	}
}

func TestModrinthResolve(t *testing.T) {
	srv := newModrinthServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/project/fabric-api/version/0.163.0" {
			t.Errorf("path = %q, want /project/fabric-api/version/0.163.0", r.URL.Path)
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing User-Agent header")
		}
		w.Header().Set("Content-Type", "application/json")
		writeResponse(t, w, `{
			"version_number": "0.163.0",
			"files": [{
				"hashes": {"sha1": "abc123", "sha512": "def456"},
				"url": "https://cdn.example.com/fabric-api-0.163.0.jar",
				"filename": "fabric-api-0.163.0.jar",
				"primary": true,
				"size": 1774707
			}]
		}`)
	})

	m := Modrinth{baseURL: srv.URL}
	art, err := m.Resolve(context.Background(), config.ModSpec{
		Name: "fabric-api", Source: "modrinth", Version: "0.163.0",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if art.Name != "fabric-api" {
		t.Errorf("Name = %q, want %q", art.Name, "fabric-api")
	}
	if art.URL != "https://cdn.example.com/fabric-api-0.163.0.jar" {
		t.Errorf("URL = %q", art.URL)
	}
	if art.Filename != "fabric-api-0.163.0.jar" {
		t.Errorf("Filename = %q", art.Filename)
	}
	if art.Size != 1774707 {
		t.Errorf("Size = %d, want 1774707", art.Size)
	}
	if art.Hashes["sha1"] != "abc123" || art.Hashes["sha512"] != "def456" {
		t.Errorf("Hashes = %v, want sha1=abc123 sha512=def456", art.Hashes)
	}
}

func TestModrinthResolvePicksPrimary(t *testing.T) {
	srv := newModrinthServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeResponse(t, w, `{
			"files": [
				{"url": "https://cdn.example.com/sources.jar", "filename": "sources.jar", "primary": false, "size": 100},
				{"url": "https://cdn.example.com/mod.jar", "filename": "mod.jar", "primary": true, "size": 200}
			]
		}`)
	})

	m := Modrinth{baseURL: srv.URL}
	art, err := m.Resolve(context.Background(), config.ModSpec{Name: "x", Source: "modrinth", Version: "1"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if art.Filename != "mod.jar" {
		t.Errorf("Filename = %q, want %q (primary file)", art.Filename, "mod.jar")
	}
}

func TestModrinthResolveFallsBackToFirstFile(t *testing.T) {
	srv := newModrinthServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeResponse(t, w, `{
			"files": [
				{"url": "https://cdn.example.com/only.jar", "filename": "only.jar", "primary": false, "size": 50}
			]
		}`)
	})

	m := Modrinth{baseURL: srv.URL}
	art, err := m.Resolve(context.Background(), config.ModSpec{Name: "x", Source: "modrinth", Version: "1"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if art.Filename != "only.jar" {
		t.Errorf("Filename = %q, want %q (fallback to first file)", art.Filename, "only.jar")
	}
}

func TestModrinthResolveNotFound(t *testing.T) {
	srv := newModrinthServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	m := Modrinth{baseURL: srv.URL}
	if _, err := m.Resolve(context.Background(), config.ModSpec{Name: "nope", Source: "modrinth", Version: "1"}); err == nil {
		t.Fatal("Resolve: expected error for 404, got nil")
	}
}

func TestModrinthResolveNoFiles(t *testing.T) {
	srv := newModrinthServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeResponse(t, w, `{"files": []}`)
	})

	m := Modrinth{baseURL: srv.URL}
	if _, err := m.Resolve(context.Background(), config.ModSpec{Name: "x", Source: "modrinth", Version: "1"}); err == nil {
		t.Fatal("Resolve: expected error for empty files, got nil")
	}
}

func TestResolveUnknownSource(t *testing.T) {
	if _, err := Resolve(context.Background(), config.ModSpec{Name: "x", Source: "unknown", Version: "1"}); err == nil {
		t.Fatal("Resolve: expected error for unknown source, got nil")
	}
}
