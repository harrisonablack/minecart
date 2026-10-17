package run

import (
	"slices"
	"testing"

	"github.com/harrisonablack/minecart/pkg/config"
)

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name string
		s    config.ServerConfig
		want []string
	}{
		{
			name: "minimal",
			s:    config.ServerConfig{},
			want: []string{"-jar", "server.jar"},
		},
		{
			name: "with memory",
			s:    config.ServerConfig{Memory: "2G"},
			want: []string{"-Xmx2G", "-jar", "server.jar"},
		},
		{
			name: "with args",
			s:    config.ServerConfig{Memory: "2G", Args: []string{"nogui"}},
			want: []string{"-Xmx2G", "-jar", "server.jar", "nogui"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildArgs(tt.s)
			if !slices.Equal(got, tt.want) {
				t.Errorf("buildArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
