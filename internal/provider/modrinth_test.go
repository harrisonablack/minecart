package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harrisonablack/minecart/internal/config"
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
	art, err := m.Resolve(context.Background(), config.ResourceSpec{
		Name: "fabric-api", Provider: "modrinth", Version: "0.163.0",
	}, "1.21.4")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if art.Name != "fabric-api" {
		t.Errorf("Name = %q, want %q", art.Name, "fabric-api")
	}
	if art.Version != "0.163.0" {
		t.Errorf("Version = %q, want %q", art.Version, "0.163.0")
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

func TestModrinthResolvePinnedFallsBackToSpecVersion(t *testing.T) {
	srv := newModrinthServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeResponse(t, w, `{
			"files": [{"url": "https://cdn.example.com/mod.jar", "filename": "mod.jar", "primary": true, "size": 100}]
		}`)
	})

	m := Modrinth{baseURL: srv.URL}
	art, err := m.Resolve(context.Background(), config.ResourceSpec{Name: "x", Provider: "modrinth", Version: "1.2.3"}, "1.21.4")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if art.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q (fall back to spec.Version)", art.Version, "1.2.3")
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
	art, err := m.Resolve(context.Background(), config.ResourceSpec{Name: "x", Provider: "modrinth", Version: "1"}, "1.21.4")
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
	art, err := m.Resolve(context.Background(), config.ResourceSpec{Name: "x", Provider: "modrinth", Version: "1"}, "1.21.4")
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
	if _, err := m.Resolve(context.Background(), config.ResourceSpec{Name: "nope", Provider: "modrinth", Version: "1"}, "1.21.4"); err == nil {
		t.Fatal("Resolve: expected error for 404, got nil")
	}
}

func TestModrinthResolveNoFiles(t *testing.T) {
	srv := newModrinthServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeResponse(t, w, `{"files": []}`)
	})

	m := Modrinth{baseURL: srv.URL}
	if _, err := m.Resolve(context.Background(), config.ResourceSpec{Name: "x", Provider: "modrinth", Version: "1"}, "1.21.4"); err == nil {
		t.Fatal("Resolve: expected error for empty files, got nil")
	}
}

func TestResolveUnknownProvider(t *testing.T) {
	if _, err := Resolve(context.Background(), config.ResourceSpec{Name: "x", Provider: "unknown", Version: "1"}, "1.21.4"); err == nil {
		t.Fatal("Resolve: expected error for unknown provider, got nil")
	}
}

func TestModrinthResolveLatest(t *testing.T) {
	srv := newModrinthServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/project/luckperms/version" {
			t.Errorf("path = %q, want /project/luckperms/version", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		// Newer version listed first but with an older date, to prove the
		// resolver sorts by date rather than trusting API ordering.
		writeResponse(t, w, `[
			{
				"id": "v1",
				"version_number": "5.3.0",
				"version_type": "release",
				"game_versions": ["1.21.4"],
				"loaders": ["paper"],
				"date_published": "2024-01-01T00:00:00Z",
				"files": [{"url": "https://cdn.example.com/lp-5.3.0.jar", "filename": "lp-5.3.0.jar", "primary": true, "size": 200}]
			},
			{
				"id": "v2",
				"version_number": "5.2.0",
				"version_type": "release",
				"game_versions": ["1.21.4"],
				"loaders": ["paper"],
				"date_published": "2024-02-01T00:00:00Z",
				"files": [{"url": "https://cdn.example.com/lp-5.2.0.jar", "filename": "lp-5.2.0.jar", "primary": true, "size": 100}]
			}
		]`)
	})

	m := Modrinth{baseURL: srv.URL}
	art, err := m.Resolve(context.Background(), config.ResourceSpec{Name: "luckperms", Provider: "modrinth"}, "1.21.4")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if art.Version != "5.2.0" {
		t.Errorf("Version = %q, want %q (newest by date)", art.Version, "5.2.0")
	}
	if art.Filename != "lp-5.2.0.jar" {
		t.Errorf("Filename = %q, want %q", art.Filename, "lp-5.2.0.jar")
	}
}

func TestModrinthResolveLatestFiltersGameVersion(t *testing.T) {
	srv := newModrinthServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeResponse(t, w, `[
			{
				"id": "v1",
				"version_number": "5.3.0",
				"version_type": "release",
				"game_versions": ["1.20.4"],
				"loaders": ["paper"],
				"date_published": "2024-02-01T00:00:00Z",
				"files": [{"url": "https://cdn.example.com/lp-5.3.0.jar", "filename": "lp-5.3.0.jar", "primary": true, "size": 200}]
			},
			{
				"id": "v2",
				"version_number": "5.2.0",
				"version_type": "release",
				"game_versions": ["1.21.4"],
				"loaders": ["paper"],
				"date_published": "2024-01-01T00:00:00Z",
				"files": [{"url": "https://cdn.example.com/lp-5.2.0.jar", "filename": "lp-5.2.0.jar", "primary": true, "size": 100}]
			}
		]`)
	})

	m := Modrinth{baseURL: srv.URL}
	art, err := m.Resolve(context.Background(), config.ResourceSpec{Name: "luckperms", Provider: "modrinth"}, "1.21.4")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if art.Version != "5.2.0" {
		t.Errorf("Version = %q, want %q (only version compatible with 1.21.4)", art.Version, "5.2.0")
	}
}

func TestModrinthResolveLatestFiltersLoader(t *testing.T) {
	srv := newModrinthServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeResponse(t, w, `[
			{
				"id": "v1",
				"version_number": "5.3.0",
				"version_type": "release",
				"game_versions": ["1.21.4"],
				"loaders": ["fabric"],
				"date_published": "2024-02-01T00:00:00Z",
				"files": [{"url": "https://cdn.example.com/lp-5.3.0.jar", "filename": "lp-5.3.0.jar", "primary": true, "size": 200}]
			},
			{
				"id": "v2",
				"version_number": "5.2.0",
				"version_type": "release",
				"game_versions": ["1.21.4"],
				"loaders": ["paper"],
				"date_published": "2024-01-01T00:00:00Z",
				"files": [{"url": "https://cdn.example.com/lp-5.2.0.jar", "filename": "lp-5.2.0.jar", "primary": true, "size": 100}]
			}
		]`)
	})

	m := Modrinth{baseURL: srv.URL}
	art, err := m.Resolve(context.Background(), config.ResourceSpec{Name: "luckperms", Provider: "modrinth"}, "1.21.4")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if art.Version != "5.2.0" {
		t.Errorf("Version = %q, want %q (only paper loader)", art.Version, "5.2.0")
	}
}

func TestModrinthResolveLatestChannel(t *testing.T) {
	versions := `[
		{"id": "r", "version_number": "1.0.0", "version_type": "release", "game_versions": ["1.21.4"], "loaders": ["paper"], "date_published": "2024-01-01T00:00:00Z", "files": [{"url": "https://cdn.example.com/r.jar", "filename": "r.jar", "primary": true, "size": 1}]},
		{"id": "b", "version_number": "1.1.0-beta.1", "version_type": "beta", "game_versions": ["1.21.4"], "loaders": ["paper"], "date_published": "2024-02-01T00:00:00Z", "files": [{"url": "https://cdn.example.com/b.jar", "filename": "b.jar", "primary": true, "size": 1}]},
		{"id": "a", "version_number": "1.2.0-alpha.1", "version_type": "alpha", "game_versions": ["1.21.4"], "loaders": ["paper"], "date_published": "2024-03-01T00:00:00Z", "files": [{"url": "https://cdn.example.com/a.jar", "filename": "a.jar", "primary": true, "size": 1}]}
	]`

	tests := []struct {
		name    string
		channel string
		want    string
	}{
		{name: "stable", channel: "stable", want: "1.0.0"},
		{name: "empty defaults to stable", channel: "", want: "1.0.0"},
		{name: "beta", channel: "beta", want: "1.1.0-beta.1"},
		{name: "alpha", channel: "alpha", want: "1.2.0-alpha.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newModrinthServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				writeResponse(t, w, versions)
			})

			m := Modrinth{baseURL: srv.URL}
			art, err := m.Resolve(context.Background(), config.ResourceSpec{Name: "x", Provider: "modrinth", Channel: tt.channel}, "1.21.4")
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if art.Version != tt.want {
				t.Errorf("Version = %q, want %q", art.Version, tt.want)
			}
		})
	}
}

func TestModrinthResolveLatestStableNoRelease(t *testing.T) {
	srv := newModrinthServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeResponse(t, w, `[
			{"id": "b", "version_number": "1.1.0-beta.1", "version_type": "beta", "game_versions": ["1.21.4"], "loaders": ["paper"], "date_published": "2024-02-01T00:00:00Z", "files": [{"url": "https://cdn.example.com/b.jar", "filename": "b.jar", "primary": true, "size": 1}]}
		]`)
	})

	m := Modrinth{baseURL: srv.URL}
	_, err := m.Resolve(context.Background(), config.ResourceSpec{Name: "x", Provider: "modrinth"}, "1.21.4")
	if err == nil {
		t.Fatal("Resolve: expected error for no stable release, got nil")
	}
	if !strings.Contains(err.Error(), "no stable release compatible with version") {
		t.Errorf("error = %q, want stable release message", err)
	}
}

func TestModrinthResolveLatestNoCompatible(t *testing.T) {
	srv := newModrinthServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeResponse(t, w, `[
			{"id": "r", "version_number": "1.0.0", "version_type": "release", "game_versions": ["1.20.4"], "loaders": ["paper"], "date_published": "2024-01-01T00:00:00Z", "files": [{"url": "https://cdn.example.com/r.jar", "filename": "r.jar", "primary": true, "size": 1}]}
		]`)
	})

	m := Modrinth{baseURL: srv.URL}
	if _, err := m.Resolve(context.Background(), config.ResourceSpec{Name: "x", Provider: "modrinth"}, "1.21.4"); err == nil {
		t.Fatal("Resolve: expected error for no compatible version, got nil")
	}
}
