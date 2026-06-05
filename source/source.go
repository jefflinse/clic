// Package source loads clic and OpenAPI spec content from local files or remote URLs.
package source

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const fetchTimeout = 30 * time.Second

// IsURL reports whether the given location is an http(s) URL.
func IsURL(location string) bool {
	return strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://")
}

// Load reads spec content from a local file path or an http(s) URL.
func Load(location string) ([]byte, error) {
	if IsURL(location) {
		return loadURL(location)
	}

	return os.ReadFile(location)
}

func loadURL(url string) ([]byte, error) {
	client := http.Client{Timeout: fetchTimeout}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: unexpected status %d", url, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
