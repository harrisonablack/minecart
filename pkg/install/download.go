package install

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/harrisonablack/minecart/pkg/artifact"
	"github.com/harrisonablack/minecart/pkg/version"
)

// Download fetches the artifact and writes it into dir, skipping when the file
// is already present and verified.
func Download(ctx context.Context, logger *slog.Logger, a artifact.Artifact, dir string) error {
	dest := filepath.Join(dir, a.Filename)

	if matches(a, dest) {
		logger.Info("already installed", "artifact", a.Name, "file", dest)
		return nil
	}

	if err := download(ctx, logger, a, dest); err != nil {
		_ = os.Remove(dest)
		return err
	}

	if err := verify(a, dest, logger); err != nil {
		_ = os.Remove(dest)
		return err
	}

	return nil
}

// download fetches a.URL and writes it to dest, logging progress.
func download(ctx context.Context, logger *slog.Logger, a artifact.Artifact, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.URL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", version.UserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	logger.Info("downloading", "artifact", a.Name, "url", a.URL)

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}

	if a.Size > 0 {
		logger.Info("downloaded", "artifact", a.Name, "bytes", n, "percent", float64(n)/float64(a.Size)*100)
	} else {
		logger.Info("downloaded", "artifact", a.Name, "bytes", n)
	}

	return nil
}
