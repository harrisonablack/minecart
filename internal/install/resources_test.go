package install

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/harrisonablack/minecart/internal/artifact"
	"github.com/harrisonablack/minecart/internal/config"
	"github.com/harrisonablack/minecart/internal/manifest"
)

// fakeResolver returns a fixed artifact for every spec, recording the MC
// version it was asked to resolve against.
type fakeResolver struct {
	version   string
	content   []byte
	requests  *int32
	serverURL string
}

func (f fakeResolver) resolve(ctx context.Context, spec config.ResourceSpec, mcVersion string) (artifact.Artifact, error) {
	if f.requests != nil {
		atomic.AddInt32(f.requests, 1)
	}
	return artifact.Artifact{
		Name:     spec.Name,
		Version:  f.version,
		URL:      f.serverURL + "/" + spec.Name + ".jar",
		Filename: spec.Name + "-" + f.version + ".jar",
		Hashes:   map[string]string{"sha256": sha256Hex(f.content)},
		Size:     int64(len(f.content)),
	}, nil
}

// stubResolver swaps resolveResource for fn and restores it on cleanup.
func stubResolver(t *testing.T, fn func(context.Context, config.ResourceSpec, string) (artifact.Artifact, error)) {
	t.Helper()
	old := resolveResource
	resolveResource = fn
	t.Cleanup(func() { resolveResource = old })
}

// writeServerManifest writes a manifest as installServer would, so
// installResources can read and preserve its server fields.
func writeServerManifest(t *testing.T, dir string) {
	t.Helper()
	m := manifest.Manifest{
		Type:        "paper",
		Identity:    "paper-1.21.4-232",
		Entry:       manifest.Entry{Type: manifest.EntryJava, Point: "paper-1.21.4-232.jar"},
		InstalledAt: time.Now().UTC(),
	}
	if err := manifest.Write(manifest.Path(dir), m); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func TestInstallResourcesDownloadsPlugins(t *testing.T) {
	content := []byte("plugin jar")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResponse(t, w, content)
	}))
	t.Cleanup(srv.Close)

	stubResolver(t, fakeResolver{version: "1.0.0", content: content, serverURL: srv.URL}.resolve)

	dir := t.TempDir()
	writeServerManifest(t, dir)

	cfg := config.Config{
		Directory: dir,
		Server:    config.ServerConfig{Type: "paper"},
		Resources: config.ResourcesConfig{
			Plugins: []config.ResourceSpec{{Name: "luckperms", Provider: "modrinth"}},
		},
	}

	if err := installResources(context.Background(), discardLogger(), cfg, "1.21.4"); err != nil {
		t.Fatalf("installResources: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "plugins", "luckperms-1.0.0.jar")); err != nil {
		t.Errorf("plugin jar should exist, stat err = %v", err)
	}

	got, err := manifest.Read(manifest.Path(dir))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if len(got.Plugins) != 1 || got.Plugins[0].Name != "luckperms" || got.Plugins[0].Version != "1.0.0" {
		t.Errorf("Plugins = %+v, want [luckperms 1.0.0]", got.Plugins)
	}
	if got.Identity != "paper-1.21.4-232" {
		t.Errorf("Identity = %q, want preserved server identity", got.Identity)
	}
}

func TestInstallResourcesSkipsWhenCurrent(t *testing.T) {
	content := []byte("plugin jar")
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		writeResponse(t, w, content)
	}))
	t.Cleanup(srv.Close)

	stubResolver(t, fakeResolver{version: "1.0.0", content: content, serverURL: srv.URL}.resolve)

	dir := t.TempDir()
	writeServerManifest(t, dir)

	cfg := config.Config{
		Directory: dir,
		Server:    config.ServerConfig{Type: "paper"},
		Resources: config.ResourcesConfig{
			Plugins: []config.ResourceSpec{{Name: "luckperms", Provider: "modrinth"}},
		},
	}

	if err := installResources(context.Background(), discardLogger(), cfg, "1.21.4"); err != nil {
		t.Fatalf("first installResources: %v", err)
	}
	if err := installResources(context.Background(), discardLogger(), cfg, "1.21.4"); err != nil {
		t.Fatalf("second installResources: %v", err)
	}

	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Errorf("download server hit %d times, want 1 (skip when current)", got)
	}
}

func TestInstallResourcesRedownloadsOnVersionChange(t *testing.T) {
	content := []byte("plugin jar")
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		writeResponse(t, w, content)
	}))
	t.Cleanup(srv.Close)

	version := "1.0.0"
	stubResolver(t, func(ctx context.Context, spec config.ResourceSpec, mcVersion string) (artifact.Artifact, error) {
		atomic.AddInt32(&requests, 1)
		return artifact.Artifact{
			Name:     spec.Name,
			Version:  version,
			URL:      srv.URL + "/" + spec.Name + ".jar",
			Filename: spec.Name + "-" + version + ".jar",
			Hashes:   map[string]string{"sha256": sha256Hex(content)},
			Size:     int64(len(content)),
		}, nil
	})

	dir := t.TempDir()
	writeServerManifest(t, dir)

	cfg := config.Config{
		Directory: dir,
		Server:    config.ServerConfig{Type: "paper"},
		Resources: config.ResourcesConfig{
			Plugins: []config.ResourceSpec{{Name: "luckperms", Provider: "modrinth"}},
		},
	}

	if err := installResources(context.Background(), discardLogger(), cfg, "1.21.4"); err != nil {
		t.Fatalf("first installResources: %v", err)
	}

	version = "2.0.0"
	if err := installResources(context.Background(), discardLogger(), cfg, "1.21.4"); err != nil {
		t.Fatalf("second installResources: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "plugins", "luckperms-2.0.0.jar")); err != nil {
		t.Errorf("new plugin jar should exist, stat err = %v", err)
	}
	got, err := manifest.Read(manifest.Path(dir))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if len(got.Plugins) != 1 || got.Plugins[0].Version != "2.0.0" {
		t.Errorf("Plugins = %+v, want version 2.0.0", got.Plugins)
	}
}

func TestInstallResourcesSyncRemovesUndeclared(t *testing.T) {
	content := []byte("plugin jar")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResponse(t, w, content)
	}))
	t.Cleanup(srv.Close)

	stubResolver(t, fakeResolver{version: "1.0.0", content: content, serverURL: srv.URL}.resolve)

	dir := t.TempDir()
	writeServerManifest(t, dir)

	both := config.Config{
		Directory: dir,
		Server:    config.ServerConfig{Type: "paper"},
		Resources: config.ResourcesConfig{
			Plugins: []config.ResourceSpec{
				{Name: "luckperms", Provider: "modrinth"},
				{Name: "viaversion", Provider: "modrinth"},
			},
		},
	}
	if err := installResources(context.Background(), discardLogger(), both, "1.21.4"); err != nil {
		t.Fatalf("install both: %v", err)
	}

	onlyLuckperms := config.Config{
		Directory: dir,
		Server:    config.ServerConfig{Type: "paper"},
		Resources: config.ResourcesConfig{
			Sync:    true,
			Plugins: []config.ResourceSpec{{Name: "luckperms", Provider: "modrinth"}},
		},
	}
	if err := installResources(context.Background(), discardLogger(), onlyLuckperms, "1.21.4"); err != nil {
		t.Fatalf("install with sync: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "plugins", "viaversion-1.0.0.jar")); !os.IsNotExist(err) {
		t.Errorf("undeclared plugin should be removed, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "plugins", "luckperms-1.0.0.jar")); err != nil {
		t.Errorf("declared plugin should remain, stat err = %v", err)
	}
	got, err := manifest.Read(manifest.Path(dir))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if len(got.Plugins) != 1 || got.Plugins[0].Name != "luckperms" {
		t.Errorf("Plugins = %+v, want [luckperms]", got.Plugins)
	}
}

func TestInstallResourcesWarnsOnVanilla(t *testing.T) {
	dir := t.TempDir()
	writeServerManifest(t, dir)

	cfg := config.Config{
		Directory: dir,
		Server:    config.ServerConfig{Type: "vanilla"},
		Resources: config.ResourcesConfig{
			Plugins: []config.ResourceSpec{{Name: "luckperms", Provider: "modrinth"}},
		},
	}

	if err := installResources(context.Background(), discardLogger(), cfg, "1.21.4"); err != nil {
		t.Fatalf("installResources: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "plugins")); !os.IsNotExist(err) {
		t.Errorf("plugins dir should not be created for vanilla, stat err = %v", err)
	}
}

func TestInstallResourcesNoPlugins(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		Directory: dir,
		Server:    config.ServerConfig{Type: "paper"},
	}
	if err := installResources(context.Background(), discardLogger(), cfg, "1.21.4"); err != nil {
		t.Fatalf("installResources: %v", err)
	}
}
