package manifest

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestReadMissing(t *testing.T) {
	m, err := Read(filepath.Join(t.TempDir(), Filename))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !reflect.DeepEqual(m, Manifest{}) {
		t.Errorf("Read() = %+v, want zero Manifest", m)
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		m    Manifest
	}{
		{
			name: "java",
			m: Manifest{
				Type:        "paper",
				Identity:    "paper-1.21.4-232",
				Entry:       Entry{Type: EntryJava, Point: "paper-1.21.4-232.jar"},
				Hashes:      map[string]string{"sha256": "abc123"},
				InstalledAt: time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "script without hashes",
			m: Manifest{
				Type:        "neoforge",
				Identity:    "neoforge-21.4.1",
				Entry:       Entry{Type: EntryScript, Point: "run.sh"},
				InstalledAt: time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := Path(t.TempDir())
			if err := Write(path, tt.m); err != nil {
				t.Fatalf("Write: %v", err)
			}
			got, err := Read(path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if !reflect.DeepEqual(got, tt.m) {
				t.Errorf("Read() = %+v, want %+v", got, tt.m)
			}
		})
	}
}

func TestWriteYAMLKeys(t *testing.T) {
	path := Path(t.TempDir())
	m := Manifest{
		Type:        "vanilla",
		Identity:    "vanilla-26.3",
		Entry:       Entry{Type: EntryJava, Point: "server.jar"},
		Hashes:      map[string]string{"sha1": "abc"},
		InstalledAt: time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC),
	}
	if err := Write(path, m); err != nil {
		t.Fatalf("Write: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	for _, key := range []string{"type:", "identity:", "entry:", "point:", "hashes:", "installed_at:"} {
		if !strings.Contains(string(data), key) {
			t.Errorf("manifest missing key %q:\n%s", key, data)
		}
	}
}

func TestReadInvalid(t *testing.T) {
	path := Path(t.TempDir())
	if err := os.WriteFile(path, []byte("type: [unterminated"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if _, err := Read(path); err == nil {
		t.Fatal("Read: expected error for invalid YAML, got nil")
	}
}
