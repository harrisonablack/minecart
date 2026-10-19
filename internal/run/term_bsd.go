//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package run

import "golang.org/x/sys/unix"

const (
	ioctlReadTermios  = unix.TIOCGETA
	ioctlWriteTermios = unix.TIOCSETA
)
