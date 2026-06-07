package oauth

import (
	"encoding/json"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

// tokenDir is where cached tokens live: ~/.clic/tokens. Files are written 0600
// and the directory 0700, since they hold bearer credentials.
func tokenDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".clic", "tokens"), nil
}

func tokenPath(key string) (string, error) {
	dir, err := tokenDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, key+".json"), nil
}

// loadToken returns the cached token for key, if one is present and readable.
func loadToken(key string) (*oauth2.Token, bool) {
	path, err := tokenPath(key)
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, false
	}
	return &tok, true
}

// saveToken persists a token for key, creating the cache directory if needed.
func saveToken(key string, tok *oauth2.Token) error {
	dir, err := tokenDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(tok)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, key+".json")
	return os.WriteFile(path, data, 0o600)
}

// removeToken deletes the cached token for key. A missing file is not an error.
func removeToken(key string) error {
	path, err := tokenPath(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
