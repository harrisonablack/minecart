package config

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/magiconair/properties"
)

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func putFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func readProperties(t *testing.T, path string) map[string]string {
	t.Helper()
	loader := properties.Loader{Encoding: properties.UTF8, DisableExpansion: true}
	p, err := loader.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return p.Map()
}

func TestApplyProperties(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.properties")
	putFile(t, path, "motd=old\ndifficulty=hard\nrcon.port=25575\ncontinued=hello\\\n  world\nliteral=${UNCHANGED}\nunicode=\\u00e9\n")
	cfg := Config{Directory: dir, Config: FileConfig{Files: map[string]map[string]any{
		"server.properties": {
			"motd":        "  Welcome\nLine two: = \\ ☃ 🚂",
			"max-players": 20,
			"online-mode": true,
			"ratio":       1.25,
			"rcon.port":   25576,
			"empty":       "",
			"expression":  "${literal}",
			"a=b:c key":   "escaped key",
		},
		"plugins/example/settings.properties": {"enabled": false},
	}}}
	if err := Apply(quietLogger(), cfg); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	want := map[string]string{
		"motd": "  Welcome\nLine two: = \\ ☃ 🚂", "max-players": "20", "online-mode": "true",
		"ratio": "1.25", "rcon.port": "25576", "empty": "", "difficulty": "hard",
		"continued": "helloworld", "literal": "${UNCHANGED}", "expression": "${literal}",
		"unicode": "é", "a=b:c key": "escaped key",
	}
	if got := readProperties(t, path); !reflect.DeepEqual(got, want) {
		t.Fatalf("properties = %#v, want %#v", got, want)
	}
	if got := readProperties(t, filepath.Join(dir, "plugins/example/settings.properties")); got["enabled"] != "false" {
		t.Fatalf("new properties = %#v", got)
	}
	first := readFile(t, path)
	if err := Apply(quietLogger(), cfg); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, readFile(t, path)) {
		t.Fatal("repeated application changed the serialized result")
	}

	// Removing a declaration leaves its stored value alone, including later
	// manual changes, while the remaining declarations are still enforced.
	delete(cfg.Config.Files["server.properties"], "motd")
	putFile(t, path, string(first)+"\nmotd=manually edited\nmax-players=5\n")
	if err := Apply(quietLogger(), cfg); err != nil {
		t.Fatal(err)
	}
	got := readProperties(t, path)
	if got["motd"] != "manually edited" || got["max-players"] != "20" {
		t.Fatalf("declaration ownership lost: %#v", got)
	}

	cfg.Config.Files = nil
	before := readFile(t, path)
	if err := Apply(quietLogger(), cfg); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, readFile(t, path)) {
		t.Fatal("omitted declarations changed an existing file")
	}
}

func TestApplyPropertiesRejectsCollectionsAndNull(t *testing.T) {
	for _, value := range []any{nil, []any{"a"}, map[string]any{"nested": true}} {
		dir := t.TempDir()
		path := filepath.Join(dir, "server.properties")
		putFile(t, path, "motd=original\n")
		cfg := Config{Directory: dir, Config: FileConfig{Files: map[string]map[string]any{
			"server.properties": {"motd": value},
		}}}
		if err := Apply(quietLogger(), cfg); err == nil || !strings.Contains(err.Error(), `property "motd"`) {
			t.Fatalf("Apply(%#v): expected property error, got %v", value, err)
		}
		if got := string(readFile(t, path)); got != "motd=original\n" {
			t.Fatalf("failed application changed destination: %q", got)
		}
	}
}

func TestApplyNumericProperties(t *testing.T) {
	for _, name := range []string{"minecart.yml", "minecart.json"} {
		t.Run(name, func(t *testing.T) {
			input := writeFile(t, name, `{
  "server": {"type": "paper", "version": "latest", "eula": true},
  "config": {"files": {"server.properties": {
    "max-world-size": 29999984,
    "large": 10000000.0,
    "small": 0.0000001,
    "ratio": 1.25
  }}}
}`)
			cfg, err := Load(input)
			if err != nil {
				t.Fatal(err)
			}
			cfg.Directory = t.TempDir()
			if err := Apply(quietLogger(), *cfg); err != nil {
				t.Fatal(err)
			}
			want := map[string]string{"max-world-size": "29999984", "large": "10000000", "small": "0.0000001", "ratio": "1.25"}
			if got := readProperties(t, filepath.Join(cfg.Directory, "server.properties")); !reflect.DeepEqual(got, want) {
				t.Fatalf("numeric properties = %#v, want %#v", got, want)
			}
		})
	}
}

func TestApplyPartialFailureAndRetry(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "b.properties")
	putFile(t, bad, `bad=\uZZZZ`)
	cfg := Config{Directory: dir, Config: FileConfig{Files: map[string]map[string]any{
		"a.properties": {"value": "new"},
		"b.properties": {"value": "new"},
	}}}
	if err := Apply(quietLogger(), cfg); err == nil || !strings.Contains(err.Error(), "b.properties") {
		t.Fatalf("Apply: expected malformed-file error, got %v", err)
	}
	if got := readProperties(t, filepath.Join(dir, "a.properties")); got["value"] != "new" {
		t.Fatalf("earlier merge lost: %#v", got)
	}
	if got := string(readFile(t, bad)); got != `bad=\uZZZZ` {
		t.Fatalf("malformed input was overwritten: %q", got)
	}
	putFile(t, bad, "retained=true\n")
	if err := Apply(quietLogger(), cfg); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if got := readProperties(t, bad); got["value"] != "new" || got["retained"] != "true" {
		t.Fatalf("retry result = %#v", got)
	}
}

func TestApplyPropertiesPreservesFileOnEncodingFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.properties")
	// This is a literal key, but the library writer emits it as a comment.
	// Our integration must report the loss instead of replacing the file.
	const original = "\\!key=retained\nmotd=original\n"
	putFile(t, path, original)
	cfg := Config{Directory: dir, Config: FileConfig{Files: map[string]map[string]any{
		"server.properties": {"motd": "new"},
	}}}
	if err := Apply(quietLogger(), cfg); err == nil || !strings.Contains(err.Error(), "without loss") {
		t.Fatalf("expected encoding failure, got %v", err)
	}
	if got := string(readFile(t, path)); got != original {
		t.Fatalf("encoding failure modified the file: %q", got)
	}
}

func TestApplyInvalidDestinations(t *testing.T) {
	for _, name := range []string{"../outside.properties", "/outside.properties", "", "settings.json"} {
		t.Run(name, func(t *testing.T) {
			cfg := Config{Directory: t.TempDir(), Config: FileConfig{Files: map[string]map[string]any{
				name: {"motd": "hello"},
			}}}
			if err := Apply(quietLogger(), cfg); err == nil {
				t.Fatal("expected destination error")
			}
		})
	}
}

func TestWriteConfigAtomicReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.properties")
	putFile(t, path, "old")
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	old, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = old.Close() }()
	if err := writeConfig(path, []byte("new")); err != nil {
		t.Fatal(err)
	}
	retained, err := io.ReadAll(old)
	if err != nil || string(retained) != "old" {
		t.Fatalf("original file was modified in place: %q, %v", retained, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("original permissions not preserved: %v, %v", info, err)
	}
	if got := string(readFile(t, path)); got != "new" {
		t.Fatalf("replacement = %q", got)
	}
}

func TestWriteConfigCleansUpFailedReplacement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.properties")
	putFile(t, filepath.Join(path, "blocker"), "keep")
	if err := writeConfig(path, []byte("new")); err == nil {
		t.Fatal("expected rename failure")
	}
	if got := string(readFile(t, filepath.Join(path, "blocker"))); got != "keep" {
		t.Fatal("failed replacement modified destination")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary file leaked: %v, %v", entries, err)
	}
}

func TestWriteConfigRespectsUmask(t *testing.T) {
	// A subprocess keeps the process-global umask isolated from other tests.
	if dir := os.Getenv("MINECART_TEST_UMASK_DIR"); dir != "" {
		mask, err := strconv.ParseUint(os.Getenv("MINECART_TEST_UMASK"), 8, 32)
		if err != nil {
			t.Fatal(err)
		}
		previous := syscall.Umask(int(mask))
		defer syscall.Umask(previous)
		for _, name := range []string{"new.properties", "existing.properties", "private.properties"} {
			path := filepath.Join(dir, name)
			exists := name != "new.properties"
			// Check the file immediately after creation, before writeConfig can
			// chmod it. A final-mode assertion alone cannot detect the exposure.
			f, err := createConfigTemp(path, exists)
			if err != nil {
				t.Fatal(err)
			}
			info, err := f.Stat()
			_ = f.Close()
			_ = os.Remove(f.Name())
			if err != nil {
				t.Fatal(err)
			}
			want := os.FileMode(0o644) &^ os.FileMode(mask)
			if exists {
				want = 0o600 &^ os.FileMode(mask)
			}
			if got := info.Mode().Perm(); got != want {
				t.Errorf("%s temporary permissions at creation = %o, want %o", name, got, want)
			}
			if err := writeConfig(path, []byte("password=secret\n")); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for mask, newMode := range map[string]os.FileMode{"022": 0o644, "077": 0o600} {
		t.Run(mask, func(t *testing.T) {
			dir := t.TempDir()
			for name, mode := range map[string]os.FileMode{"existing.properties": 0o640, "private.properties": 0o600} {
				path := filepath.Join(dir, name)
				putFile(t, path, "old=value\n")
				if err := os.Chmod(path, mode); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.CommandContext(t.Context(), exe, "-test.run=^TestWriteConfigRespectsUmask$")
			cmd.Env = append(os.Environ(), "MINECART_TEST_UMASK_DIR="+dir, "MINECART_TEST_UMASK="+mask)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("umask subprocess: %v\n%s", err, out)
			}
			for name, mode := range map[string]os.FileMode{"new.properties": newMode, "existing.properties": 0o640, "private.properties": 0o600} {
				path := filepath.Join(dir, name)
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				if got := info.Mode().Perm(); got != mode {
					t.Errorf("%s permissions = %o, want %o", name, got, mode)
				}
				if got := string(readFile(t, path)); got != "password=secret\n" {
					t.Errorf("%s contents = %q", name, got)
				}
			}
		})
	}
}
