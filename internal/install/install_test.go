package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/harrisonablack/minecart/internal/artifact"
	"github.com/harrisonablack/minecart/internal/manifest"
	"github.com/harrisonablack/minecart/internal/platform"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func writeResponse(t *testing.T, w http.ResponseWriter, body []byte) {
	t.Helper()
	if _, err := w.Write(body); err != nil {
		t.Errorf("write response: %v", err)
	}
}

// paperResolved builds a resolved paper server for build served at srvURL.
func paperResolved(srvURL string, build int, content []byte) platform.Resolved {
	filename := fmt.Sprintf("paper-1.21.4-%d.jar", build)
	return platform.Resolved{
		Artifact: artifact.Artifact{
			Name:     "paper",
			URL:      srvURL + "/" + filename,
			Filename: filename,
			Hashes:   map[string]string{"sha256": sha256Hex(content)},
			Size:     int64(len(content)),
		},
		Entry:    manifest.Entry{Type: manifest.EntryJava, Point: filename},
		Identity: fmt.Sprintf("paper-1.21.4-%d", build),
	}
}

func TestInstallServerWritesManifest(t *testing.T) {
	content := []byte("build 52")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResponse(t, w, content)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	resolved := paperResolved(srv.URL, 52, content)
	if err := installServer(context.Background(), discardLogger(), "paper", resolved, dir); err != nil {
		t.Fatalf("installServer: %v", err)
	}

	m, err := manifest.Read(manifest.Path(dir))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if m.Type != "paper" {
		t.Errorf("Type = %q, want %q", m.Type, "paper")
	}
	if m.Identity != resolved.Identity {
		t.Errorf("Identity = %q, want %q", m.Identity, resolved.Identity)
	}
	if m.Entry != resolved.Entry {
		t.Errorf("Entry = %+v, want %+v", m.Entry, resolved.Entry)
	}
	if m.Hashes["sha256"] != resolved.Artifact.Hashes["sha256"] {
		t.Errorf("Hashes = %v, want %v", m.Hashes, resolved.Artifact.Hashes)
	}
	if m.InstalledAt.IsZero() {
		t.Error("InstalledAt is zero")
	}
	if _, err := os.Stat(filepath.Join(dir, "paper-1.21.4-52.jar")); err != nil {
		t.Errorf("jar should exist, stat err = %v", err)
	}
}

func TestInstallServerPreservesPlugins(t *testing.T) {
	content := []byte("build 53")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResponse(t, w, content)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()

	// Seed a manifest with a plugin, as a previous install would have.
	seed := manifest.Manifest{
		Type:        "paper",
		Identity:    "paper-1.21.4-52",
		Entry:       manifest.Entry{Type: manifest.EntryJava, Point: "paper-1.21.4-52.jar"},
		InstalledAt: time.Now().UTC(),
		Plugins: []manifest.Plugin{
			{Name: "luckperms", Provider: "modrinth", Version: "1.0.0", Filename: "luckperms-1.0.0.jar"},
		},
	}
	if err := manifest.Write(manifest.Path(dir), seed); err != nil {
		t.Fatalf("write seed manifest: %v", err)
	}

	resolved := paperResolved(srv.URL, 53, content)
	if err := installServer(context.Background(), discardLogger(), "paper", resolved, dir); err != nil {
		t.Fatalf("installServer: %v", err)
	}

	got, err := manifest.Read(manifest.Path(dir))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if got.Identity != resolved.Identity {
		t.Errorf("Identity = %q, want %q", got.Identity, resolved.Identity)
	}
	if len(got.Plugins) != 1 || got.Plugins[0].Name != "luckperms" {
		t.Errorf("Plugins = %+v, want [luckperms] preserved", got.Plugins)
	}
}

func TestInstallServerUpdateDetection(t *testing.T) {
	builds := map[string][]byte{
		"/paper-1.21.4-52.jar": []byte("build 52"),
		"/paper-1.21.4-53.jar": []byte("build 53"),
	}

	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		body, ok := builds[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeResponse(t, w, body)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	logger := discardLogger()
	ctx := context.Background()

	build52 := paperResolved(srv.URL, 52, builds["/paper-1.21.4-52.jar"])
	build53 := paperResolved(srv.URL, 53, builds["/paper-1.21.4-53.jar"])

	steps := []struct {
		name         string
		resolved     platform.Resolved
		wantRequests int32
		wantPresent  string
		wantAbsent   string
	}{
		{name: "fresh install", resolved: build52, wantRequests: 1, wantPresent: "paper-1.21.4-52.jar"},
		{name: "same identity skips download", resolved: build52, wantRequests: 1, wantPresent: "paper-1.21.4-52.jar"},
		{name: "new build upgrades", resolved: build53, wantRequests: 2, wantPresent: "paper-1.21.4-53.jar", wantAbsent: "paper-1.21.4-52.jar"},
		{name: "upgraded is current", resolved: build53, wantRequests: 2, wantPresent: "paper-1.21.4-53.jar"},
	}

	for _, step := range steps {
		if err := installServer(ctx, logger, "paper", step.resolved, dir); err != nil {
			t.Fatalf("%s: installServer: %v", step.name, err)
		}
		if got := atomic.LoadInt32(&requests); got != step.wantRequests {
			t.Errorf("%s: server hit %d times, want %d", step.name, got, step.wantRequests)
		}
		if _, err := os.Stat(filepath.Join(dir, step.wantPresent)); err != nil {
			t.Errorf("%s: %s should exist, stat err = %v", step.name, step.wantPresent, err)
		}
		if step.wantAbsent != "" {
			if _, err := os.Stat(filepath.Join(dir, step.wantAbsent)); !os.IsNotExist(err) {
				t.Errorf("%s: %s should be removed, stat err = %v", step.name, step.wantAbsent, err)
			}
		}
		m, err := manifest.Read(manifest.Path(dir))
		if err != nil {
			t.Fatalf("%s: read manifest: %v", step.name, err)
		}
		if m.Identity != step.resolved.Identity {
			t.Errorf("%s: manifest Identity = %q, want %q", step.name, m.Identity, step.resolved.Identity)
		}
	}
}

func TestInstallServerRedownloadsCorruptFile(t *testing.T) {
	content := []byte("build 52")
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		writeResponse(t, w, content)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	logger := discardLogger()
	resolved := paperResolved(srv.URL, 52, content)

	if err := installServer(context.Background(), logger, "paper", resolved, dir); err != nil {
		t.Fatalf("first installServer: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, resolved.Artifact.Filename), []byte("corrupt"), 0o644); err != nil {
		t.Fatalf("corrupt jar: %v", err)
	}
	if err := installServer(context.Background(), logger, "paper", resolved, dir); err != nil {
		t.Fatalf("second installServer: %v", err)
	}

	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Errorf("server hit %d times, want 2 (re-download corrupt file)", got)
	}
}

func TestInstallServerKeepsOldJarOnManifestWriteFailure(t *testing.T) {
	builds := map[string][]byte{
		"/paper-1.21.4-52.jar": []byte("build 52"),
		"/paper-1.21.4-53.jar": []byte("build 53"),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := builds[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeResponse(t, w, body)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	logger := discardLogger()
	ctx := context.Background()

	build52 := paperResolved(srv.URL, 52, builds["/paper-1.21.4-52.jar"])
	build53 := paperResolved(srv.URL, 53, builds["/paper-1.21.4-53.jar"])

	if err := installServer(ctx, logger, "paper", build52, dir); err != nil {
		t.Fatalf("install build 52: %v", err)
	}

	// A directory at the temporary path makes manifest.Write fail.
	blocker := manifest.Path(dir) + ".tmp"
	if err := os.Mkdir(blocker, 0o755); err != nil {
		t.Fatalf("create blocker: %v", err)
	}

	if err := installServer(ctx, logger, "paper", build53, dir); err == nil {
		t.Fatal("install build 53: expected error from manifest write, got nil")
	}

	oldJar := filepath.Join(dir, "paper-1.21.4-52.jar")
	if _, err := os.Stat(oldJar); err != nil {
		t.Errorf("old jar should survive failed manifest write, stat err = %v", err)
	}
	m, err := manifest.Read(manifest.Path(dir))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if m.Identity != build52.Identity {
		t.Errorf("manifest Identity = %q, want %q", m.Identity, build52.Identity)
	}

	if err := os.Remove(blocker); err != nil {
		t.Fatalf("remove blocker: %v", err)
	}
	if err := installServer(ctx, logger, "paper", build53, dir); err != nil {
		t.Fatalf("retry install build 53: %v", err)
	}
	if m, err = manifest.Read(manifest.Path(dir)); err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if m.Identity != build53.Identity {
		t.Errorf("manifest Identity = %q, want %q", m.Identity, build53.Identity)
	}
	if _, err := os.Stat(oldJar); !os.IsNotExist(err) {
		t.Errorf("old jar should be removed after successful update, stat err = %v", err)
	}
}
