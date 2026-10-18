package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/harrisonablack/minecart/internal/artifact"
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

func TestDownload(t *testing.T) {
	content := []byte("hello world")
	digest := sha256Hex(content)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResponse(t, w, content)
	}))
	t.Cleanup(srv.Close)

	a := artifact.Artifact{
		Name:     "test",
		URL:      srv.URL + "/file.jar",
		Filename: "file.jar",
		Hashes:   map[string]string{"sha256": digest},
		Size:     int64(len(content)),
	}

	dir := t.TempDir()
	if err := Download(context.Background(), discardLogger(), a, dir); err != nil {
		t.Fatalf("Download: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "file.jar"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}
}

func TestDownloadIdempotent(t *testing.T) {
	content := []byte("hello world")
	digest := sha256Hex(content)

	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		writeResponse(t, w, content)
	}))
	t.Cleanup(srv.Close)

	a := artifact.Artifact{
		Name:     "test",
		URL:      srv.URL + "/file.jar",
		Filename: "file.jar",
		Hashes:   map[string]string{"sha256": digest},
		Size:     int64(len(content)),
	}

	dir := t.TempDir()
	logger := discardLogger()

	if err := Download(context.Background(), logger, a, dir); err != nil {
		t.Fatalf("first Download: %v", err)
	}
	if err := Download(context.Background(), logger, a, dir); err != nil {
		t.Fatalf("second Download: %v", err)
	}

	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Errorf("server hit %d times, want 1 (idempotent)", got)
	}
}

func TestDownloadHashMismatch(t *testing.T) {
	content := []byte("hello world")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResponse(t, w, content)
	}))
	t.Cleanup(srv.Close)

	a := artifact.Artifact{
		Name:     "test",
		URL:      srv.URL + "/file.jar",
		Filename: "file.jar",
		Hashes:   map[string]string{"sha256": "deadbeef"},
		Size:     int64(len(content)),
	}

	dir := t.TempDir()
	if err := Download(context.Background(), discardLogger(), a, dir); err == nil {
		t.Fatal("Download: expected error for hash mismatch, got nil")
	}

	// The corrupt file should be removed.
	if _, err := os.Stat(filepath.Join(dir, "file.jar")); !os.IsNotExist(err) {
		t.Errorf("corrupt file should be removed, stat err = %v", err)
	}
}

func TestDownloadNoHash(t *testing.T) {
	content := []byte("hello world")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResponse(t, w, content)
	}))
	t.Cleanup(srv.Close)

	a := artifact.Artifact{
		Name:     "test",
		URL:      srv.URL + "/file.jar",
		Filename: "file.jar",
	}

	dir := t.TempDir()
	if err := Download(context.Background(), discardLogger(), a, dir); err != nil {
		t.Fatalf("Download: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "file.jar")); err != nil {
		t.Errorf("file should exist after download, stat err = %v", err)
	}
}

func TestVerify(t *testing.T) {
	content := []byte("hello world")
	digest := sha256Hex(content)

	dir := t.TempDir()
	path := filepath.Join(dir, "file.jar")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	tests := []struct {
		name    string
		hashes  map[string]string
		wantErr bool
	}{
		{
			name:    "no hash",
			hashes:  nil,
			wantErr: false,
		},
		{
			name:    "no supported hash",
			hashes:  map[string]string{"sha3-256": "abc"},
			wantErr: false,
		},
		{
			name:    "match",
			hashes:  map[string]string{"sha256": digest},
			wantErr: false,
		},
		{
			name:    "mismatch",
			hashes:  map[string]string{"sha256": "deadbeef"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := artifact.Artifact{Name: "test", Hashes: tt.hashes}
			err := verify(a, path, discardLogger())
			if (err != nil) != tt.wantErr {
				t.Errorf("verify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStrongestHash(t *testing.T) {
	tests := []struct {
		name       string
		hashes     map[string]string
		wantAlgo   string
		wantDigest string
	}{
		{
			name:       "empty",
			hashes:     nil,
			wantAlgo:   "",
			wantDigest: "",
		},
		{
			name:       "sha1 only",
			hashes:     map[string]string{"sha1": "a"},
			wantAlgo:   "sha1",
			wantDigest: "a",
		},
		{
			name:       "prefers sha512 over sha1",
			hashes:     map[string]string{"sha1": "a", "sha512": "b"},
			wantAlgo:   "sha512",
			wantDigest: "b",
		},
		{
			name:       "prefers sha256 over sha512",
			hashes:     map[string]string{"sha256": "c", "sha512": "b"},
			wantAlgo:   "sha256",
			wantDigest: "c",
		},
		{
			name:       "unsupported only",
			hashes:     map[string]string{"sha3-256": "e"},
			wantAlgo:   "",
			wantDigest: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			algo, digest := strongestHash(tt.hashes)
			if algo != tt.wantAlgo || digest != tt.wantDigest {
				t.Errorf("strongestHash() = (%q, %q), want (%q, %q)", algo, digest, tt.wantAlgo, tt.wantDigest)
			}
		})
	}
}

func TestWriteEULA(t *testing.T) {
	dir := t.TempDir()
	if err := writeEULA(dir); err != nil {
		t.Fatalf("writeEULA: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "eula.txt"))
	if err != nil {
		t.Fatalf("read eula.txt: %v", err)
	}
	if string(got) != "eula=true\n" {
		t.Errorf("eula.txt = %q, want %q", got, "eula=true\n")
	}
}
