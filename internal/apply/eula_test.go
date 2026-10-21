package apply

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/harrisonablack/minecart/internal/config"
)

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

func TestRunWritesEULAAndConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		Directory: dir,
		Config: config.FileConfig{Files: map[string]map[string]any{
			"server.properties": {"motd": "hello"},
		}},
	}

	if err := Run(quietLogger(), cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := string(readFile(t, filepath.Join(dir, "eula.txt"))); got != "eula=true\n" {
		t.Errorf("eula.txt = %q, want %q", got, "eula=true\n")
	}
	if got := readProperties(t, filepath.Join(dir, "server.properties")); got["motd"] != "hello" {
		t.Errorf("server.properties motd = %q, want %q", got["motd"], "hello")
	}
}

func TestRunEULAWriteFailureStopsConfig(t *testing.T) {
	dir := t.TempDir()
	// A directory at eula.txt makes writeEULA's WriteFile fail, while the
	// directory itself remains writable so Apply would otherwise succeed.
	if err := os.Mkdir(filepath.Join(dir, "eula.txt"), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Directory: dir,
		Config: config.FileConfig{Files: map[string]map[string]any{
			"server.properties": {"motd": "hello"},
		}},
	}

	if err := Run(quietLogger(), cfg); err == nil {
		t.Fatal("Run: expected error from eula write, got nil")
	}

	// Apply must not run: the declared file is not written.
	if _, err := os.Stat(filepath.Join(dir, "server.properties")); !os.IsNotExist(err) {
		t.Errorf("server.properties should not exist after eula failure, stat err = %v", err)
	}
}
