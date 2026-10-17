// Package run launches the Minecraft server process.
package run

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/harrisonablack/minecart/pkg/config"
)

// Server launches the server as a child process, forwarding stdio and OS
// signals, and blocks until it exits.
func Server(ctx context.Context, logger *slog.Logger, cfg config.Config) error {
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
			logger.Info("forwarding signal to server", "signal", sig.String())
			if err := cmd.Process.Signal(sig); err != nil {
				logger.Warn("failed to forward signal", "signal", sig.String(), "error", err)
			}
		case <-ctx.Done():
			logger.Info("stopping server")
			_ = cmd.Process.Signal(syscall.SIGTERM)
			return <-done
		case err := <-done:
			return err
		}
	}
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
