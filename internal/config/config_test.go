package config

import (
	"os"
	"path/filepath"
	"slices"
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
mods:
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
	if len(cfg.Mods) != 1 {
		t.Fatalf("len(Mods) = %d, want 1", len(cfg.Mods))
	}
	if cfg.Mods[0].Name != "fabric-api" {
		t.Errorf("Mods[0].Name = %q, want %q", cfg.Mods[0].Name, "fabric-api")
	}
	if cfg.Mods[0].Provider != "modrinth" {
		t.Errorf("Mods[0].Provider = %q, want %q", cfg.Mods[0].Provider, "modrinth")
	}
}

func TestLoadJSON(t *testing.T) {
	path := writeFile(t, "minecart.json", `{
  "server": {"type": "fabric", "version": "1.21.4", "eula": true},
  "mods": [{"name": "lithium", "provider": "modrinth", "version": "0.14.0"}]
}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Type != "fabric" {
		t.Errorf("Server.Type = %q, want %q", cfg.Server.Type, "fabric")
	}
	if len(cfg.Mods) != 1 || cfg.Mods[0].Name != "lithium" {
		t.Errorf("Mods = %v, want [lithium]", cfg.Mods)
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
			name: "mod missing name",
			cfg: Config{
				Server: valid,
				Mods:   []ModSpec{{Provider: "modrinth"}},
			},
			wantErr: true,
		},
		{
			name: "mod missing provider",
			cfg: Config{
				Server: valid,
				Mods:   []ModSpec{{Name: "lithium"}},
			},
			wantErr: true,
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
