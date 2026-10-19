package platform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harrisonablack/minecart/internal/config"
	"github.com/harrisonablack/minecart/internal/manifest"
)

// writeResponse writes body to w, failing the test on error.
func writeResponse(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if _, err := w.Write([]byte(body)); err != nil {
		t.Errorf("write response: %v", err)
	}
}

// newVanillaServer serves a manifest at /manifest and a version at /version,
// with the manifest's version URL pointing back at the test server. The
// manifest's latest release is 26.3.
func newVanillaServer(t *testing.T, versionJSON string) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest":
			writeResponse(t, w, `{
				"latest": {"release": "26.3", "snapshot": "26.4-snapshot-1"},
				"versions": [
					{"id": "26.4-snapshot-1", "url": "`+srv.URL+`/snapshot"},
					{"id": "26.3", "url": "`+srv.URL+`/version"}
				]
			}`)
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
	res, err := v.resolve(context.Background(), "26.3")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	art := res.Artifact

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
	wantEntry := manifest.Entry{Type: "java", Point: "server.jar"}
	if res.Entry != wantEntry {
		t.Errorf("Entry = %+v, want %+v", res.Entry, wantEntry)
	}
	if res.Identity != "vanilla-26.3" {
		t.Errorf("Identity = %q, want %q", res.Identity, "vanilla-26.3")
	}
}

func TestResolveVanillaVersionNotFound(t *testing.T) {
	srv := newVanillaServer(t, `{}`)

	v := vanilla{manifestURL: srv.URL + "/manifest"}
	if _, err := v.resolve(context.Background(), "1.0.0"); err == nil {
		t.Fatal("resolve: expected error for unknown version, got nil")
	}
}

func TestResolveVanillaLatest(t *testing.T) {
	srv := newVanillaServer(t, `{
		"downloads": {"server": {
			"sha1": "abc123",
			"size": 62294556,
			"url": "https://piston-data.mojang.com/v1/objects/abc123/server.jar"
		}}
	}`)

	v := vanilla{manifestURL: srv.URL + "/manifest"}
	res, err := v.resolve(context.Background(), LatestVersion)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if res.Artifact.URL != "https://piston-data.mojang.com/v1/objects/abc123/server.jar" {
		t.Errorf("URL = %q", res.Artifact.URL)
	}
	if res.Identity != "vanilla-26.3" {
		t.Errorf("Identity = %q, want %q", res.Identity, "vanilla-26.3")
	}
}

func TestResolveVanillaLatestMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResponse(t, w, `{"latest": {"release": ""}, "versions": []}`)
	}))
	t.Cleanup(srv.Close)

	v := vanilla{manifestURL: srv.URL}
	if _, err := v.resolve(context.Background(), LatestVersion); err == nil {
		t.Fatal("resolve: expected error for missing latest release, got nil")
	}
}

func TestResolveUnknownType(t *testing.T) {
	_, err := Resolve(context.Background(), config.ServerConfig{Type: "fabric", Version: "1.21.4"})
	if err == nil {
		t.Fatal("Resolve: expected error for unknown type, got nil")
	}
}

func TestFilenameFromURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name: "vanilla",
			url:  "https://piston-data.mojang.com/v1/objects/abc123/server.jar",
			want: "server.jar",
		},
		{
			name: "paper",
			url:  "https://fill-data.papermc.io/v1/objects/abc123/paper-1.21.4-232.jar",
			want: "paper-1.21.4-232.jar",
		},
		{
			name: "strips query and fragment",
			url:  "https://example.com/files/server.jar?token=x#frag",
			want: "server.jar",
		},
		{
			name:    "no path",
			url:     "https://example.com",
			wantErr: true,
		},
		{
			name:    "root path",
			url:     "https://example.com/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filenameFromURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Fatalf("filenameFromURL() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("filenameFromURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
