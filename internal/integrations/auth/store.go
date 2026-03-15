// Package auth implements session storage using OS keychain with protected-file fallback.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// FileStore persists sessions as a protected JSON file.
// It is used as the fallback when no OS keychain is available.
type FileStore struct {
	path string
}

// NewFileStore creates a FileStore at the given path.
// If path is empty it uses the XDG state directory.
func NewFileStore(path string) (*FileStore, error) {
	if path == "" {
		var err error
		path, err = defaultSessionPath()
		if err != nil {
			return nil, err
		}
	}
	return &FileStore{path: path}, nil
}

type persistedSession struct {
	Cookies []persistedCookie `json:"cookies"`
}

type persistedCookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Load reads a saved session from the file. Returns (nil, nil) if the file
// does not exist.
func (s *FileStore) Load(_ context.Context) (*Session, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading session file: %w", err)
	}

	var ps persistedSession
	if err := json.Unmarshal(data, &ps); err != nil {
		return nil, fmt.Errorf("parsing session file: %w", err)
	}

	sess := &Session{}
	for _, c := range ps.Cookies {
		sess.Cookies = append(sess.Cookies, Cookie{Name: c.Name, Value: c.Value})
	}
	return sess, nil
}

// Save writes a session to a protected file (mode 0600).
func (s *FileStore) Save(_ context.Context, sess *Session) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("creating session directory: %w", err)
	}

	ps := persistedSession{}
	for _, c := range sess.Cookies {
		ps.Cookies = append(ps.Cookies, persistedCookie{Name: c.Name, Value: c.Value})
	}

	data, err := json.Marshal(ps)
	if err != nil {
		return fmt.Errorf("encoding session: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("writing session file: %w", err)
	}
	return nil
}

// Clear removes the session file.
func (s *FileStore) Clear(_ context.Context) error {
	err := os.Remove(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func defaultSessionPath() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("finding home directory: %w", err)
		}
		switch runtime.GOOS {
		case "darwin":
			base = filepath.Join(home, "Library", "Application Support")
		default:
			base = filepath.Join(home, ".local", "state")
		}
	}
	return filepath.Join(base, "spotpilot", "session.json"), nil
}
