// Package manifest records what server was installed into a data directory and
// how to launch it. The manifest is the source of truth for both launching and
// update detection, and is only rewritten when the server files change.
package manifest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"
)

// Filename is the name of the manifest file within the data directory.
const Filename = ".manifest"

// Entry types.
const (
	EntryJava   = "java"
	EntryScript = "script"
)

// Entry describes how to launch the installed server.
type Entry struct {
	Type  string `yaml:"type"`  // "java" | "script"
	Point string `yaml:"point"` // jar filename (java) or script filename (script)
}

// Manifest records an installed server.
type Manifest struct {
	Type        string            `yaml:"type"`     // platform type: vanilla | paper | ...
	Identity    string            `yaml:"identity"` // opaque, platform-defined stable id
	Entry       Entry             `yaml:"entry"`
	Hashes      map[string]string `yaml:"hashes,omitempty"` // algorithm -> hex digest
	InstalledAt time.Time         `yaml:"installed_at"`
}

// Path returns the manifest path within dir.
func Path(dir string) string {
	return filepath.Join(dir, Filename)
}

// Read reads the manifest at path. A missing file yields a zero Manifest and a
// nil error.
func Read(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Manifest{}, nil
		}
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest %q: %w", path, err)
	}
	return m, nil
}

// Write writes m to path, replacing any existing manifest atomically.
func Write(path string, m Manifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}
