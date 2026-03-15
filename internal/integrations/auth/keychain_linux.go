//go:build linux

package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// secretToolAttrs are the attribute key/value pairs that uniquely identify the
// spotpilot credential in the secret-service D-Bus store.
var secretToolAttrs = []string{"service", "spotpilot", "account", "session"}

type keychainStore struct{}

// newKeychainStore returns a keychainStore if secret-tool is installed,
// or an error to trigger the file-store fallback.
func newKeychainStore() (CredentialStore, error) {
	if _, err := exec.LookPath("secret-tool"); err != nil {
		return nil, errors.New("secret-tool not found; falling back to file store")
	}
	return &keychainStore{}, nil
}

func (k *keychainStore) Load(_ context.Context) (*Session, error) {
	out, err := exec.Command("secret-tool",
		append([]string{"lookup"}, secretToolAttrs...)...).Output()
	if err != nil {
		// Item not found or secret-service unavailable — not an error.
		return nil, nil
	}
	data := strings.TrimSpace(string(out))
	if data == "" {
		return nil, nil
	}
	var ps persistedSession
	if err := json.Unmarshal([]byte(data), &ps); err != nil {
		// Corrupted entry — treat as no session.
		return nil, nil
	}
	sess := &Session{}
	for _, c := range ps.Cookies {
		sess.Cookies = append(sess.Cookies, Cookie{Name: c.Name, Value: c.Value})
	}
	return sess, nil
}

func (k *keychainStore) Save(_ context.Context, sess *Session) error {
	ps := persistedSession{}
	for _, c := range sess.Cookies {
		ps.Cookies = append(ps.Cookies, persistedCookie{Name: c.Name, Value: c.Value})
	}
	data, err := json.Marshal(ps)
	if err != nil {
		return fmt.Errorf("encoding session: %w", err)
	}
	args := append([]string{"store", "--label=spotpilot session"}, secretToolAttrs...)
	cmd := exec.Command("secret-tool", args...)
	cmd.Stdin = bytes.NewReader(data)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("writing to secret-service: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (k *keychainStore) Clear(_ context.Context) error {
	args := append([]string{"clear"}, secretToolAttrs...)
	// secret-tool clear exits 0 even when no matching item exists.
	return exec.Command("secret-tool", args...).Run()
}
