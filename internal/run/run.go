// Package run launches the Minecraft server process.
package run

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/harrisonablack/minecart/internal/config"
	"github.com/harrisonablack/minecart/internal/java"
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

	args := buildArgs(cfg.Server)

	cmd := exec.Command("java", args...)
	cmd.Dir = cfg.Directory
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logger.Info("starting server", "command", "java "+strings.Join(args, " "))

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	for {
		select {
		case sig := <-sigCh:
			// The vanilla server responds to SIGTERM but ignores SIGINT when
			// not attached to a terminal, so translate any stop signal to
			// SIGTERM.
			logger.Info("forwarding signal to server", "signal", sig.String())
			if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
				logger.Warn("failed to forward signal", "signal", sig.String(), "error", err)
			}
		case <-ctx.Done():
			logger.Info("stopping server")
			_ = cmd.Process.Signal(syscall.SIGTERM)
			return exitError(<-done)
		case err := <-done:
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

// buildArgs builds the java command-line arguments for the server.
func buildArgs(s config.ServerConfig) []string {
	var args []string
	if s.Memory != "" {
		args = append(args, "-Xmx"+s.Memory)
	}
	args = append(args, "-jar", "server.jar")
	args = append(args, s.Args...)
	return args
}
