package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeInDir creates an empty file named name in dir.
func writeInDir(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(""), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestDiscoverEnvVar(t *testing.T) {
	t.Setenv("MINECART_CONFIG", "/custom/config.yml")

	path, err := Discover(t.TempDir())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if path != "/custom/config.yml" {
		t.Errorf("Discover() = %q, want %q", path, "/custom/config.yml")
	}
}

func TestDiscoverYML(t *testing.T) {
	t.Setenv("MINECART_CONFIG", "")
	dir := t.TempDir()
	writeInDir(t, dir, "minecart.yml")

	path, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if want := filepath.Join(dir, "minecart.yml"); path != want {
		t.Errorf("Discover() = %q, want %q", path, want)
	}
}

func TestDiscoverYAML(t *testing.T) {
	t.Setenv("MINECART_CONFIG", "")
	dir := t.TempDir()
	writeInDir(t, dir, "minecart.yaml")

	path, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if want := filepath.Join(dir, "minecart.yaml"); path != want {
		t.Errorf("Discover() = %q, want %q", path, want)
	}
}

func TestDiscoverJSON(t *testing.T) {
	t.Setenv("MINECART_CONFIG", "")
	dir := t.TempDir()
	writeInDir(t, dir, "minecart.json")

	path, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if want := filepath.Join(dir, "minecart.json"); path != want {
		t.Errorf("Discover() = %q, want %q", path, want)
	}
}

func TestDiscoverPrecedence(t *testing.T) {
	t.Setenv("MINECART_CONFIG", "")
	dir := t.TempDir()
	writeInDir(t, dir, "minecart.json")
	writeInDir(t, dir, "minecart.yml")

	path, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if want := filepath.Join(dir, "minecart.yml"); path != want {
		t.Errorf("Discover() = %q, want %q (yml takes precedence)", path, want)
	}
}

func TestDiscoverNone(t *testing.T) {
	t.Setenv("MINECART_CONFIG", "")

	if _, err := Discover(t.TempDir()); err == nil {
		t.Fatal("Discover: expected error for no config file, got nil")
	}
}

func TestLoadDefault(t *testing.T) {
	path := writeFile(t, "minecart.yml", `
server:
  type: vanilla
  version: "1.21.4"
  eula: true
`)
	t.Setenv("MINECART_CONFIG", path)

	cfg, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault: %v", err)
	}
	if cfg.Server.Type != "vanilla" {
		t.Errorf("Server.Type = %q, want %q", cfg.Server.Type, "vanilla")
	}
}
