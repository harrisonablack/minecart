//go:build unix

package run

import (
	"testing"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

func TestMakeRawKeepsSignalsAndOutputProcessing(t *testing.T) {
	ptmx, tty, err := pty.Open()
	if err != nil {
		t.Skipf("open pty: %v", err)
	}
	defer func() { _ = ptmx.Close() }()
	defer func() { _ = tty.Close() }()

	fd := int(tty.Fd())
	if !isTerminal(fd) {
		t.Fatal("pty slave not detected as terminal")
	}

	// A fresh pty already has VMIN=1/VTIME=0, the values makeRaw sets, so
	// seed distinct values to prove restore puts them back.
	seed, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		t.Fatal(err)
	}
	seed.Cc[unix.VMIN] = 3
	seed.Cc[unix.VTIME] = 7
	if err := unix.IoctlSetTermios(fd, ioctlWriteTermios, seed); err != nil {
		t.Fatal(err)
	}

	before, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		t.Fatal(err)
	}
	if before.Lflag&unix.ICANON == 0 || before.Lflag&unix.ECHO == 0 {
		t.Fatal("expected fresh pty to start in canonical echo mode")
	}
	if before.Cc[unix.VMIN] != 3 || before.Cc[unix.VTIME] != 7 {
		t.Fatalf("seed VMIN/VTIME = %d/%d, want 3/7", before.Cc[unix.VMIN], before.Cc[unix.VTIME])
	}

	state, err := makeRaw(fd)
	if err != nil {
		t.Fatalf("makeRaw: %v", err)
	}

	raw, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		got  uint64
		on   bool
	}{
		{"ICANON", uint64(raw.Lflag & unix.ICANON), false},
		{"ECHO", uint64(raw.Lflag & unix.ECHO), false},
		{"IEXTEN", uint64(raw.Lflag & unix.IEXTEN), false},
		{"ICRNL", uint64(raw.Iflag & unix.ICRNL), false},
		{"IXON", uint64(raw.Iflag & unix.IXON), false},
		{"ISIG", uint64(raw.Lflag & unix.ISIG), true},
		{"OPOST", uint64(raw.Oflag & unix.OPOST), true},
	} {
		if (c.got != 0) != c.on {
			t.Errorf("%s: on=%v, want on=%v", c.name, c.got != 0, c.on)
		}
	}
	if raw.Cc[unix.VMIN] != 1 || raw.Cc[unix.VTIME] != 0 {
		t.Errorf("VMIN/VTIME = %d/%d, want 1/0", raw.Cc[unix.VMIN], raw.Cc[unix.VTIME])
	}

	if err := state.restore(fd); err != nil {
		t.Fatalf("restore: %v", err)
	}
	after, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		t.Fatal(err)
	}
	if *after != *before {
		t.Errorf("restore did not return terminal to original state:\n got %+v\nwant %+v", *after, *before)
	}
}
