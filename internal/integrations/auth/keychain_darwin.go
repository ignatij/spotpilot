//go:build darwin

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const (
	keychainService = "spotpilot"
	keychainAccount = "spotpilot-session"
)

type keychainStore struct{}

// newKeychainStore returns a keychainStore. On macOS the security CLI is
// always present, so this never fails.
func newKeychainStore() (CredentialStore, error) {
	return &keychainStore{}, nil
}

func (k *keychainStore) Load(_ context.Context) (*Session, error) {
	out, err := exec.Command("security", "find-generic-password",
		"-a", keychainAccount, "-s", keychainService, "-w").Output()
	if err != nil {
		// A non-zero exit (e.g. item not found) is not a store error.
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading from keychain: %w", err)
	}
	data := strings.TrimSpace(string(out))
	var ps persistedSession
	if err := json.Unmarshal([]byte(data), &ps); err != nil {
		// Corrupted or unrecognised entry — treat as no session.
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
	// Delete any existing entry first to allow idempotent saves.
	_ = exec.Command("security", "delete-generic-password",
		"-a", keychainAccount, "-s", keychainService).Run()

	if out, err := exec.Command("security", "add-generic-password",
		"-a", keychainAccount, "-s", keychainService, "-w", string(data)).CombinedOutput(); err != nil {
		return fmt.Errorf("writing to keychain: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (k *keychainStore) Clear(_ context.Context) error {
	err := exec.Command("security", "delete-generic-password",
		"-a", keychainAccount, "-s", keychainService).Run()
	// A non-zero exit simply means the item was not present — that is fine.
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return nil
	}
	return err
}
