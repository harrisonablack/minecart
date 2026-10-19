// Package run launches the Minecraft server process.
package run

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/creack/pty"
	"github.com/harrisonablack/minecart/internal/config"
	"github.com/harrisonablack/minecart/internal/java"
	"github.com/harrisonablack/minecart/internal/manifest"
)

// Server launches the server as a child process, forwarding stdio and OS
// signals, and blocks until it exits.
func Server(ctx context.Context, logger *slog.Logger, cfg config.Config) error {
	info, err := java.Detect(ctx)
	if err != nil {
		return fmt.Errorf("detect java: %w", err)
	}
	attrs := []any{"version", info.Version}
	if info.Vendor != "" {
		attrs = append(attrs, "vendor", info.Vendor)
	}
	logger.Info("resolved java", attrs...)

	m, err := manifest.Read(manifest.Path(cfg.Directory))
	if err != nil {
		return err
	}
	if m.Entry.Type == "" {
		return fmt.Errorf("no installed server in %q: manifest missing", cfg.Directory)
	}

	name, args, err := buildCommand(m.Entry, cfg.Server)
	if err != nil {
		return err
	}
	if m.Entry.Type == manifest.EntryScript && (cfg.Server.Memory != "" || len(cfg.Server.JvmFlags) > 0) {
		logger.Warn("memory and jvm_flags are ignored for script entries")
	}

	// exec.Command rather than exec.CommandContext: the latter SIGKILLs the
	// child on cancellation, which would skip the server's world save.
	cmd := exec.Command(name, args...)
	cmd.Dir = cfg.Directory

	logger.Info("starting server", "entry", m.Entry.Type, "command", name+" "+strings.Join(args, " "))

	// The server's console (e.g. Paper's JLine) puts its pty into raw mode
	// and handles echo and line editing itself. Our terminal must stop
	// doing the same, or every keystroke is echoed twice and escape
	// sequences for arrow keys and pastes show up as text. Do this before
	// launching the server so a failure here leaves no child to clean up.
	stdinFd := int(os.Stdin.Fd())
	interactive := isTerminal(stdinFd)
	if interactive {
		state, err := makeRaw(stdinFd)
		if err != nil {
			return fmt.Errorf("configure terminal: %w", err)
		}
		defer func() { _ = state.restore(stdinFd) }()
	}

	// Run the server in its own session with a pseudo-terminal so that
	// terminal-generated signals (ctrl-c) reach only minecart, not the
	// server. This keeps shutdown behavior consistent across servers that
	// install their own SIGINT handler (Paper) and those that don't
	// (Vanilla): minecart is the sole controller of shutdown.
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("start server: %w", err)
	}
	defer func() { _ = ptmx.Close() }()

	if interactive {
		// Match the child's terminal size to the real terminal up front. A
		// pty starts at 0x0, which makes console programs fall back to a
		// "dumb" terminal and print "..." prompts.
		_ = pty.InheritSize(os.Stdin, ptmx)

		// Keep the child's terminal size in sync with the real terminal so
		// console output stays correct on resize.
		winch := make(chan os.Signal, 1)
		signal.Notify(winch, syscall.SIGWINCH)
		defer signal.Stop(winch)
		stopResize := make(chan struct{})
		defer close(stopResize)
		go func() {
			for {
				select {
				case <-winch:
					_ = pty.InheritSize(os.Stdin, ptmx)
				case <-stopResize:
					return
				}
			}
		}()
	} else {
		_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})
	}

	// Relay the server's output to our stdout and our stdin to the server.
	// The output relay ends once the child exits and closes its side of the
	// pty; waiting on it ensures the server's final lines are not dropped.
	outputDone := make(chan struct{})
	go func() {
		defer close(outputDone)
		_, _ = io.Copy(os.Stdout, ptmx)
	}()
	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	for {
		select {
		case sig := <-sigCh:
			// Only minecart receives terminal signals now, so always send
			// the server SIGTERM: it triggers a graceful stop on every
			// server type, whereas vanilla ignores SIGINT.
			logger.Info("forwarding signal to server", "signal", sig.String())
			if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
				logger.Warn("failed to forward signal", "signal", sig.String(), "error", err)
			}
		case <-ctx.Done():
			logger.Info("stopping server")
			_ = cmd.Process.Signal(syscall.SIGTERM)
			err := <-done
			<-outputDone
			return exitError(err)
		case err := <-done:
			<-outputDone
			return exitError(err)
		}
	}
}

// exitError returns nil when the child exited normally or stopped cleanly
// after a signal we forwarded, otherwise the exit error.
func exitError(err error) error {
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return err
	}
	// The JVM exits with 128+signal after a graceful shutdown triggered by
	// SIGTERM/SIGINT, so treat those as a clean stop.
	switch exitErr.ExitCode() {
	case 128 + int(syscall.SIGTERM), 128 + int(syscall.SIGINT):
		return nil
	}
	return err
}

// buildCommand builds the command name and arguments that launch entry.
// Memory and JvmFlags only apply to java entries; script entries manage their
// own JVM arguments.
func buildCommand(entry manifest.Entry, s config.ServerConfig) (string, []string, error) {
	if entry.Point == "" {
		return "", nil, fmt.Errorf("%s entry has no entry point", entry.Type)
	}

	var args []string
	switch entry.Type {
	case manifest.EntryJava:
		args = append(args, s.JvmFlags...)
		if s.Memory != "" {
			args = append(args, "-Xmx"+s.Memory)
		}
		args = append(args, "-jar", entry.Point)
		args = append(args, s.Args...)
		return "java", args, nil
	case manifest.EntryScript:
		args = append(args, entry.Point)
		args = append(args, s.Args...)
		return "sh", args, nil
	default:
		return "", nil, fmt.Errorf("unknown entry type %q", entry.Type)
	}
}
