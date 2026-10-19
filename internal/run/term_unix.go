//go:build unix

package run

import "golang.org/x/sys/unix"

// termState holds terminal attributes saved by makeRaw so they can be
// restored once the server exits.
type termState struct {
	termios unix.Termios
}

// isTerminal reports whether fd refers to a terminal.
func isTerminal(fd int) bool {
	_, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	return err == nil
}

// makeRaw turns off line buffering, echo and extended editing keys on fd so
// that keystrokes reach the server's pty untouched and the server's console
// is the only thing echoing input. Unlike a full raw mode, signal keys and
// output post-processing stay on: ctrl-c must keep interrupting minecart (not
// the server), and minecart's own log lines must still render correctly.
func makeRaw(fd int) (*termState, error) {
	old, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		return nil, err
	}

	raw := *old
	raw.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	raw.Lflag &^= unix.ICANON | unix.ECHO | unix.IEXTEN
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(fd, ioctlWriteTermios, &raw); err != nil {
		return nil, err
	}
	return &termState{termios: *old}, nil
}

// restore puts fd back into the state saved by makeRaw.
func (s *termState) restore(fd int) error {
	return unix.IoctlSetTermios(fd, ioctlWriteTermios, &s.termios)
}
