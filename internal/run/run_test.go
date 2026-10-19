package run

import (
	"slices"
	"testing"

	"github.com/harrisonablack/minecart/internal/config"
	"github.com/harrisonablack/minecart/internal/manifest"
)

func TestBuildCommand(t *testing.T) {
	vanilla := manifest.Entry{Type: manifest.EntryJava, Point: "server.jar"}
	paper := manifest.Entry{Type: manifest.EntryJava, Point: "paper-1.21.4-232.jar"}
	script := manifest.Entry{Type: manifest.EntryScript, Point: "run.sh"}

	tests := []struct {
		name     string
		entry    manifest.Entry
		s        config.ServerConfig
		wantName string
		want     []string
	}{
		{
			name:     "java minimal",
			entry:    vanilla,
			s:        config.ServerConfig{},
			wantName: "java",
			want:     []string{"-jar", "server.jar"},
		},
		{
			name:     "java with memory",
			entry:    vanilla,
			s:        config.ServerConfig{Memory: "2G"},
			wantName: "java",
			want:     []string{"-Xmx2G", "-jar", "server.jar"},
		},
		{
			name:     "java with args",
			entry:    vanilla,
			s:        config.ServerConfig{Memory: "2G", Args: []string{"nogui"}},
			wantName: "java",
			want:     []string{"-Xmx2G", "-jar", "server.jar", "nogui"},
		},
		{
			name:     "java uses entry point",
			entry:    paper,
			s:        config.ServerConfig{Args: []string{"nogui"}},
			wantName: "java",
			want:     []string{"-jar", "paper-1.21.4-232.jar", "nogui"},
		},
		{
			name:  "java with jvm flags in order",
			entry: paper,
			s: config.ServerConfig{
				Memory:   "4G",
				JvmFlags: []string{"-Xms4G", "-XX:+UseG1GC", "-XX:MaxGCPauseMillis=200"},
				Args:     []string{"nogui"},
			},
			wantName: "java",
			want: []string{
				"-Xms4G", "-XX:+UseG1GC", "-XX:MaxGCPauseMillis=200",
				"-Xmx4G", "-jar", "paper-1.21.4-232.jar", "nogui",
			},
		},
		{
			name:     "java with jvm flags and no memory",
			entry:    vanilla,
			s:        config.ServerConfig{JvmFlags: []string{"-XX:+UseZGC"}},
			wantName: "java",
			want:     []string{"-XX:+UseZGC", "-jar", "server.jar"},
		},
		{
			name:     "script minimal",
			entry:    script,
			s:        config.ServerConfig{},
			wantName: "sh",
			want:     []string{"run.sh"},
		},
		{
			name:  "script ignores memory and jvm flags",
			entry: script,
			s: config.ServerConfig{
				Memory:   "4G",
				JvmFlags: []string{"-XX:+UseG1GC"},
				Args:     []string{"nogui"},
			},
			wantName: "sh",
			want:     []string{"run.sh", "nogui"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, got, err := buildCommand(tt.entry, tt.s)
			if err != nil {
				t.Fatalf("buildCommand: %v", err)
			}
			if name != tt.wantName {
				t.Errorf("buildCommand() name = %q, want %q", name, tt.wantName)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("buildCommand() args = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildCommandInvalid(t *testing.T) {
	tests := []struct {
		name  string
		entry manifest.Entry
	}{
		{name: "unknown type", entry: manifest.Entry{Type: "argfile", Point: "args.txt"}},
		{name: "empty point", entry: manifest.Entry{Type: manifest.EntryJava}},
		{name: "zero entry", entry: manifest.Entry{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := buildCommand(tt.entry, config.ServerConfig{}); err == nil {
				t.Fatal("buildCommand: expected error, got nil")
			}
		})
	}
}
