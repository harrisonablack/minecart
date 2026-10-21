package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/knadh/koanf/parsers/yaml"
)

func readYAML(t *testing.T, path string) map[string]any {
	t.Helper()
	out, err := yaml.Parser().Unmarshal(readFile(t, path))
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestLoadAndApplyMixedConfigurations(t *testing.T) {
	dir := t.TempDir()
	server := filepath.Join(dir, "server")
	path := filepath.Join(server, "config/paper-global.yml")
	putFile(t, path, `proxies:
  velocity:
    enabled: false
    secret: retained
query.port: old-literal
query:
  port: old-nested
untouched:
  child: keep
`)
	declarations := filepath.Join(dir, "minecart.yml")
	putFile(t, declarations, fmt.Sprintf(`directory: %q
server: {type: paper, version: latest, eula: true}
config:
  files:
    server.properties:
      motd: My server
      rcon.port: 25575
    config/paper-global.yml:
      proxies:
        velocity:
          enabled: true
          online-mode: true
      query.port: literal
      query:
        port: nested
    plugins/example.config/settings.with.dots.yaml:
      isolated.key: [one, two]
`, server))
	cfg, err := Load(declarations)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(quietLogger(), *cfg); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"proxies": map[string]any{"velocity": map[string]any{
			"enabled": true, "online-mode": true, "secret": "retained",
		}},
		"query.port": "literal",
		"query":      map[string]any{"port": "nested"},
		"untouched":  map[string]any{"child": "keep"},
	}
	if got := readYAML(t, path); !reflect.DeepEqual(got, want) {
		t.Fatalf("merged YAML = %#v, want %#v", got, want)
	}
	other := filepath.Join(server, "plugins/example.config/settings.with.dots.yaml")
	if got := readYAML(t, other); !reflect.DeepEqual(got, map[string]any{"isolated.key": []any{"one", "two"}}) {
		t.Fatalf("missing YAML file not created independently: %#v", got)
	}
	if got := readProperties(t, filepath.Join(server, "server.properties")); got["motd"] != "My server" || got["rcon.port"] != "25575" {
		t.Fatalf("properties in mixed declaration = %#v", got)
	}
	first := readFile(t, path)
	if err := Apply(quietLogger(), *cfg); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, readFile(t, path)) {
		t.Fatal("repeated YAML application changed the serialized result")
	}

	delete(cfg.Config.Files["config/paper-global.yml"], "query.port")
	putFile(t, path, strings.ReplaceAll(string(first), "query.port: literal", "query.port: edited"))
	if err := Apply(quietLogger(), *cfg); err != nil {
		t.Fatal(err)
	}
	if got := readYAML(t, path)["query.port"]; got != "edited" {
		t.Fatalf("removed declaration still managed: %v", got)
	}
}

func TestYAMLMergeContract(t *testing.T) {
	// Exercise the complete Minecart boundary: declarations loaded from YAML,
	// existing files merged with Koanf, then serialized and read back.
	tests := []struct {
		name, existing, declared, want string
	}{
		{"maps merge", "{keep: old, replace: old}", "{replace: new}", "{keep: old, replace: new}"},
		{"lists replace", "[old, retained]", "[new]", "[new]"},
		{"list elements are not merged", "[{keep: old, replace: old}]", "[{replace: new}]", "[{replace: new}]"},
		{"empty list clears", "[old]", "[]", "[]"},
		{"null replaces scalar", "old", "null", "null"},
		{"null replaces map", "{keep: old}", "null", "null"},
		{"scalar replaces map", "{keep: old}", "new", "new"},
		{"map replaces scalar", "old", "{new: true}", "{new: true}"},
		{"map replaces null", "null", "{new: true}", "{new: true}"},
		{"list replaces map", "{keep: old}", "[new]", "[new]"},
		{"map replaces list", "[old]", "{new: true}", "{new: true}"},
		{"empty map preserves map", "{keep: old}", "{}", "{keep: old}"},
		{"empty map replaces scalar", "old", "{}", "{}"},
		{"scalar type changes", "42", "false", "false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "settings.yaml")
			putFile(t, path, "retained: yes\nvalue: "+tt.existing+"\n")
			input := writeFile(t, "minecart.yml", fmt.Sprintf(`directory: %q
server: {type: paper, version: latest, eula: true}
config:
  files:
    settings.yaml:
      value: %s
      added.null: null
`, dir, tt.declared))
			cfg, err := Load(input)
			if err != nil {
				t.Fatal(err)
			}
			if err := Apply(quietLogger(), *cfg); err != nil {
				t.Fatal(err)
			}
			want, err := yaml.Parser().Unmarshal([]byte("retained: yes\nvalue: " + tt.want + "\nadded.null: null\n"))
			if err != nil {
				t.Fatal(err)
			}
			if got := readYAML(t, path); !reflect.DeepEqual(got, want) {
				t.Fatalf("merged YAML = %#v, want %#v", got, want)
			}
		})
	}
}

func TestApplyEmptyYAML(t *testing.T) {
	for _, content := range []string{"", "# empty\n", "null\n", "{}\n"} {
		t.Run(content, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "settings.yml")
			putFile(t, path, content)
			cfg := Config{Directory: dir, Config: FileConfig{Files: map[string]map[string]any{
				"settings.yml": {"enabled": true},
			}}}
			if err := Apply(quietLogger(), cfg); err != nil {
				t.Fatal(err)
			}
			if got := readYAML(t, path); !reflect.DeepEqual(got, map[string]any{"enabled": true}) {
				t.Fatalf("empty input merge = %#v", got)
			}
		})
	}
}

func TestYAMLInputErrorsPreserveDestination(t *testing.T) {
	for _, content := range []string{
		"broken: [", "key: first\nkey: duplicate\n", "[a, b]", "scalar",
		"key: keep\n---\nother: keep\n", "key: keep\n---\nbroken: [",
	} {
		t.Run(content, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "settings.yml")
			putFile(t, path, content)
			cfg := Config{Directory: dir, Config: FileConfig{Files: map[string]map[string]any{
				"settings.yml": {"enabled": true},
			}}}
			if err := Apply(quietLogger(), cfg); err == nil || !strings.Contains(err.Error(), "settings.yml") {
				t.Fatalf("expected YAML input error, got %v", err)
			}
			if got := string(readFile(t, path)); got != content {
				t.Fatalf("invalid input replaced: %q", got)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 {
				t.Fatalf("failed merge left temporary files: %v, %v", entries, err)
			}
		})
	}
}
