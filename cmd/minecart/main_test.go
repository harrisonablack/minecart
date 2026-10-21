package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harrisonablack/minecart/internal/manifest"
	"github.com/magiconair/properties"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Run the real entry point in a subprocess, including its os.Exit paths. Only
// external HTTP and Java are replaced; installation, merging and launch are real.
func TestMinecartProcess(t *testing.T) {
	if os.Getenv("MINECART_TEST_PROCESS") != "1" {
		return
	}
	const artifact = "test server artifact"
	digest := sha256.Sum256([]byte(artifact))
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body string
		switch r.URL.String() {
		case "https://fill.papermc.io/v3/projects/paper/versions/1.21.4/builds":
			body = fmt.Sprintf(`[{"id":7,"channel":"STABLE","downloads":{"server:default":{
"url":"https://downloads.example.test/paper.jar","checksums":{"sha256":"%x"},"size":%d}}}]`, digest, len(artifact))
		case "https://downloads.example.test/paper.jar":
			body = artifact
		default:
			return nil, fmt.Errorf("unexpected HTTP request: %s", r.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	main()
}

func setupMinecart(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "bin", "java"), `#!/bin/sh
if [ "$1" = "-XshowSettings:properties" ]; then
  printf 'java.version = 21.0.11\njava.vm.vendor = Test\n'
  exit 0
fi
if [ -f server.properties ]; then
  cp server.properties launched.properties
fi
printf 'launched\n' >> launches
`)
	if err := os.Chmod(filepath.Join(dir, "bin", "java"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeDeclarations(t *testing.T, dir, files string) {
	t.Helper()
	writeTestFile(t, filepath.Join(dir, "minecart.yml"), `directory: server
server: {type: paper, version: "1.21.4", eula: true}
config:
  files:
`+files)
}

func invokeMinecart(t *testing.T, dir string) (string, error) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestMinecartProcess$")
	cmd.Dir = dir
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "MINECART_") && !strings.HasPrefix(entry, "PATH=") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "MINECART_TEST_PROCESS=1",
		"MINECART_CONFIG="+filepath.Join(dir, "minecart.yml"),
		"PATH="+filepath.Join(dir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestConfigurationAppliedBeforeEveryLaunch(t *testing.T) {
	dir := setupMinecart(t)
	writeDeclarations(t, dir, "    server.properties: {motd: first, max-players: 20}\n")
	if out, err := invokeMinecart(t, dir); err != nil {
		t.Fatalf("first launch: %v\n%s", err, out)
	}
	server := filepath.Join(dir, "server")
	manifestBefore := readTestFile(t, manifest.Path(server))
	loader := properties.Loader{Encoding: properties.UTF8, DisableExpansion: true}
	launched, err := loader.LoadFile(filepath.Join(server, "launched.properties"))
	if err != nil || launched.GetString("motd", "") != "first" || launched.GetString("max-players", "") != "20" {
		t.Fatalf("server did not see declarations on first launch: %v, %v", launched, err)
	}

	writeTestFile(t, filepath.Join(server, "server.properties"), "motd=manual\nmax-players=5\ndifficulty=hard\n")
	writeDeclarations(t, dir, "    server.properties: {motd: second, max-players: 30}\n")
	out, err := invokeMinecart(t, dir)
	if err != nil || !strings.Contains(out, "already current") {
		t.Fatalf("current installation launch: %v\n%s", err, out)
	}
	if !bytes.Equal(manifestBefore, readTestFile(t, manifest.Path(server))) {
		t.Fatal("configuration edit changed the installation manifest")
	}
	launched, err = loader.LoadFile(filepath.Join(server, "launched.properties"))
	if err != nil || launched.GetString("motd", "") != "second" || launched.GetString("max-players", "") != "30" || launched.GetString("difficulty", "") != "hard" {
		t.Fatalf("server did not see merged edits: %v, %v", launched, err)
	}
	if got := string(readTestFile(t, filepath.Join(server, "launches"))); got != "launched\nlaunched\n" {
		t.Fatalf("launches = %q", got)
	}
}

func TestConfigurationFailurePreventsLaunch(t *testing.T) {
	for _, failure := range []string{"malformed", "read error", "write error"} {
		t.Run(failure, func(t *testing.T) {
			dir := setupMinecart(t)
			server := filepath.Join(dir, "server")
			name := "server.properties"
			if failure == "write error" {
				if os.Geteuid() == 0 {
					t.Skip("root bypasses directory write permissions")
				}
				name = "config/settings.properties"
			}
			path := filepath.Join(server, name)
			switch failure {
			case "malformed":
				writeTestFile(t, path, `motd=\uZZZZ`)
			case "read error":
				if err := os.MkdirAll(path, 0o755); err != nil {
					t.Fatal(err)
				}
			case "write error":
				writeTestFile(t, path, "motd=original\n")
				parent := filepath.Dir(path)
				if err := os.Chmod(parent, 0o555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })
			}
			writeDeclarations(t, dir, fmt.Sprintf("    %s: {motd: new}\n", name))
			out, err := invokeMinecart(t, dir)
			if err == nil || !strings.Contains(out, "apply configuration") || !strings.Contains(out, name) {
				t.Fatalf("expected configuration failure: %v\n%s", err, out)
			}
			if _, err := os.Stat(filepath.Join(server, "launches")); !os.IsNotExist(err) {
				t.Fatalf("server launched after configuration failure: %v", err)
			}
			// Installation is already committed, even though configuration failed.
			before := readTestFile(t, manifest.Path(server))
			if failure == "malformed" {
				if got := string(readTestFile(t, path)); got != `motd=\uZZZZ` {
					t.Fatalf("malformed file replaced: %q", got)
				}
				writeTestFile(t, path, "difficulty=hard\n")
				if out, err := invokeMinecart(t, dir); err != nil {
					t.Fatalf("retry launch: %v\n%s", err, out)
				}
				if !bytes.Equal(before, readTestFile(t, manifest.Path(server))) {
					t.Fatal("retry changed the committed installation")
				}
			}
			if failure == "write error" && string(readTestFile(t, path)) != "motd=original\n" {
				t.Fatal("write failure modified the existing file")
			}
		})
	}
}
