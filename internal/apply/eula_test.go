package apply

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteEULA(t *testing.T) {
	dir := t.TempDir()
	if err := writeEULA(dir); err != nil {
		t.Fatalf("writeEULA: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "eula.txt"))
	if err != nil {
		t.Fatalf("read eula.txt: %v", err)
	}
	if string(got) != "eula=true\n" {
		t.Errorf("eula.txt = %q, want %q", got, "eula=true\n")
	}
}
