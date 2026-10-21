package config

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestLoadFileDeclarations(t *testing.T) {
	for _, name := range []string{"minecart.yml", "minecart.json"} {
		t.Run(name, func(t *testing.T) {
			// JSON is also YAML: exercise both application parsers with the same
			// declaration, including literal and nested keys that flatten alike.
			path := writeFile(t, name, `{
  "server": {"type": "paper", "version": "latest", "eula": true, "memory": "2G"},
  "config": {"files": {
    "server.properties": {"rcon.port": 25575, "motd": "", "online-mode": true},
    "plugins/example.config.yml": {
      "query.port": "literal", "query": {"port": "nested"}, "nullable": null
    }
  }}
}`)
			t.Setenv("MINECART_CONFIG", path)
			t.Setenv("MINECART_SERVER_MEMORY", "4G")
			cfg, err := LoadDefault()
			if err != nil {
				t.Fatalf("LoadDefault: %v", err)
			}
			if cfg.Directory != "server" || cfg.Server.Memory != "4G" {
				t.Fatalf("defaults/env overrides lost: %+v", cfg)
			}
			props := cfg.Config.Files["server.properties"]
			if len(props) != 3 || props["motd"] != "" || props["online-mode"] != true {
				t.Fatalf("properties declaration = %#v", props)
			}
			// JSON numbers are float64, while YAML retains integers; both must be
			// preserved as numbers (formatting is tested in the apply package).
			switch props["rcon.port"].(type) {
			case int, float64:
			default:
				t.Fatalf("literal rcon.port = %#v, want a number", props["rcon.port"])
			}
			want := map[string]any{
				"query.port": "literal",
				"query":      map[string]any{"port": "nested"},
				"nullable":   nil,
			}
			if got := cfg.Config.Files["plugins/example.config.yml"]; !reflect.DeepEqual(got, want) {
				t.Fatalf("nested declaration = %#v, want %#v", got, want)
			}
		})
	}
}

func TestLoadRejectsNonMappingDeclarations(t *testing.T) {
	for _, section := range []string{
		"config: text",
		"config: []",
		"config: {files: []}",
		"config: {files: [{server.properties: {motd: hello}}]}",
		"config: {files: {server.properties: null}}",
		"config: {files: {server.properties: [{motd: hello}]}}",
		"config: {files: {server.properties: text}}",
	} {
		t.Run(section, func(t *testing.T) {
			path := writeFile(t, "minecart.yml", "server: {type: paper, version: latest, eula: true}\n"+section)
			if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "must be a mapping") {
				t.Fatalf("Load: expected mapping error, got %v", err)
			}
		})
	}
}

func TestLoadYAML(t *testing.T) {
	path := writeFile(t, "minecart.yml", `
directory: server
server:
  type: paper
  version: "1.21.4"
  eula: true
  memory: 2G
  jvm_flags: ["-Xms2G", "-XX:+UseG1GC"]
  args: ["-nogui"]
resources:
  plugins:
    - name: fabric-api
      provider: modrinth
      version: "0.115.0"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Directory != "server" {
		t.Errorf("Directory = %q, want %q", cfg.Directory, "server")
	}
	if cfg.Server.Type != "paper" {
		t.Errorf("Server.Type = %q, want %q", cfg.Server.Type, "paper")
	}
	if cfg.Server.Version != "1.21.4" {
		t.Errorf("Server.Version = %q, want %q", cfg.Server.Version, "1.21.4")
	}
	if !cfg.Server.Eula {
		t.Error("Server.Eula = false, want true")
	}
	if cfg.Server.Memory != "2G" {
		t.Errorf("Server.Memory = %q, want %q", cfg.Server.Memory, "2G")
	}
	if !slices.Equal(cfg.Server.JvmFlags, []string{"-Xms2G", "-XX:+UseG1GC"}) {
		t.Errorf("Server.JvmFlags = %v, want [-Xms2G -XX:+UseG1GC]", cfg.Server.JvmFlags)
	}
	if len(cfg.Server.Args) != 1 || cfg.Server.Args[0] != "-nogui" {
		t.Errorf("Server.Args = %v, want [-nogui]", cfg.Server.Args)
	}
	if len(cfg.Resources.Plugins) != 1 {
		t.Fatalf("len(Resources.Plugins) = %d, want 1", len(cfg.Resources.Plugins))
	}
	if cfg.Resources.Plugins[0].Name != "fabric-api" {
		t.Errorf("Resources.Plugins[0].Name = %q, want %q", cfg.Resources.Plugins[0].Name, "fabric-api")
	}
	if cfg.Resources.Plugins[0].Provider != "modrinth" {
		t.Errorf("Resources.Plugins[0].Provider = %q, want %q", cfg.Resources.Plugins[0].Provider, "modrinth")
	}
}

func TestLoadJSON(t *testing.T) {
	path := writeFile(t, "minecart.json", `{
  "server": {"type": "fabric", "version": "1.21.4", "eula": true},
  "resources": {"plugins": [{"name": "lithium", "provider": "modrinth", "version": "0.14.0"}]}
}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Type != "fabric" {
		t.Errorf("Server.Type = %q, want %q", cfg.Server.Type, "fabric")
	}
	if len(cfg.Resources.Plugins) != 1 || cfg.Resources.Plugins[0].Name != "lithium" {
		t.Errorf("Resources.Plugins = %v, want [lithium]", cfg.Resources.Plugins)
	}
}

func TestLoadResources(t *testing.T) {
	path := writeFile(t, "minecart.yml", `
directory: server
server:
  type: paper
  version: "1.21.4"
  eula: true
resources:
  sync: true
  plugins:
    - name: luckperms
      provider: modrinth
    - name: viaversion
      provider: modrinth
      version: 5.2.1
      channel: beta
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Resources.Sync {
		t.Error("Resources.Sync = false, want true")
	}
	if len(cfg.Resources.Plugins) != 2 {
		t.Fatalf("len(Resources.Plugins) = %d, want 2", len(cfg.Resources.Plugins))
	}
	if cfg.Resources.Plugins[0].Name != "luckperms" || cfg.Resources.Plugins[0].Version != "" {
		t.Errorf("Plugins[0] = %+v, want luckperms with empty version", cfg.Resources.Plugins[0])
	}
	if cfg.Resources.Plugins[1].Name != "viaversion" || cfg.Resources.Plugins[1].Version != "5.2.1" || cfg.Resources.Plugins[1].Channel != "beta" {
		t.Errorf("Plugins[1] = %+v, want viaversion 5.2.1 beta", cfg.Resources.Plugins[1])
	}
}

func TestLoadEnvOverridesFile(t *testing.T) {
	path := writeFile(t, "minecart.yml", `
server:
  type: paper
  version: "1.21.4"
  eula: true
  memory: 2G
`)

	t.Setenv("MINECART_SERVER_MEMORY", "4G")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Memory != "4G" {
		t.Errorf("Server.Memory = %q, want %q (env should override file)", cfg.Server.Memory, "4G")
	}
	if cfg.Server.Type != "paper" {
		t.Errorf("Server.Type = %q, want %q (file value should remain)", cfg.Server.Type, "paper")
	}
}

func TestLoadUnsupportedFormat(t *testing.T) {
	path := writeFile(t, "minecart.toml", "server = {}")

	if _, err := Load(path); err == nil {
		t.Fatal("Load: expected error for unsupported format, got nil")
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.yml")); err == nil {
		t.Fatal("Load: expected error for missing file, got nil")
	}
}

func TestValidate(t *testing.T) {
	valid := ServerConfig{Type: "paper", Version: "1.21.4", Eula: true}

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid",
			cfg:     Config{Server: valid},
			wantErr: false,
		},
		{
			name:    "missing type",
			cfg:     Config{Server: ServerConfig{Version: "1.21.4", Eula: true}},
			wantErr: true,
		},
		{
			name:    "missing version",
			cfg:     Config{Server: ServerConfig{Type: "paper", Eula: true}},
			wantErr: true,
		},
		{
			name:    "missing eula",
			cfg:     Config{Server: ServerConfig{Type: "paper", Version: "1.21.4"}},
			wantErr: true,
		},
		{
			name: "plugin missing name",
			cfg: Config{
				Server:    valid,
				Resources: ResourcesConfig{Plugins: []ResourceSpec{{Provider: "modrinth"}}},
			},
			wantErr: true,
		},
		{
			name: "plugin missing provider",
			cfg: Config{
				Server:    valid,
				Resources: ResourcesConfig{Plugins: []ResourceSpec{{Name: "lithium"}}},
			},
			wantErr: true,
		},
		{
			name: "plugin invalid channel",
			cfg: Config{
				Server:    valid,
				Resources: ResourcesConfig{Plugins: []ResourceSpec{{Name: "lithium", Provider: "modrinth", Channel: "nightly"}}},
			},
			wantErr: true,
		},
		{
			name: "plugin valid channel",
			cfg: Config{
				Server:    valid,
				Resources: ResourcesConfig{Plugins: []ResourceSpec{{Name: "lithium", Provider: "modrinth", Channel: "beta"}}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadDirectoryDefault(t *testing.T) {
	path := writeFile(t, "minecart.yml", `
server:
  type: vanilla
  version: "1.21.4"
  eula: true
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Directory != "server" {
		t.Errorf("Directory = %q, want default %q", cfg.Directory, "server")
	}
}

func TestLoadDirectoryOverride(t *testing.T) {
	path := writeFile(t, "minecart.yml", `
directory: /srv/minecraft
server:
  type: vanilla
  version: "1.21.4"
  eula: true
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Directory != "/srv/minecraft" {
		t.Errorf("Directory = %q, want %q", cfg.Directory, "/srv/minecraft")
	}
}
