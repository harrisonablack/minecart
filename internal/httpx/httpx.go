// Package httpx provides shared HTTP helpers.
package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/harrisonablack/minecart/internal/version"
)

// GetJSON performs a GET request and decodes the JSON response into out.
func GetJSON(ctx context.Context, client *http.Client, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", version.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
