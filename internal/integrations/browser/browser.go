// Package browser implements the BrowserLauncher port for Chrome/Chromium detection and launch.
package browser

import (
	"context"
	"errors"
	"os/exec"
)

// ErrNoBrowser is returned when neither Chrome nor Chromium is installed.
var ErrNoBrowser = errors.New("neither Google Chrome nor Chromium is installed")

// Launcher implements the app.BrowserLauncher port.
type Launcher struct{}

// New creates a new Launcher.
func New() *Launcher { return &Launcher{} }

// LaunchLogin opens url in Chrome or Chromium (preference: Chrome > Chromium).
// Returns ErrNoBrowser if neither is found.
func (l *Launcher) LaunchLogin(ctx context.Context, url string) error {
	path, err := findBrowser()
	if err != nil {
		return err
	}
	return exec.CommandContext(ctx, path, url).Start()
}

// LaunchURL opens url in the system default browser.
func (l *Launcher) LaunchURL(ctx context.Context, url string) error {
	return openDefault(ctx, url)
}

// FindChromeBinary returns the path to Chrome or Chromium, preferring Chrome.
// Returns ErrNoBrowser if neither is installed. Used by integrations that need
// to launch Chrome directly (e.g. the CDP login performer).
func FindChromeBinary() (string, error) {
	return findBrowser()
}

// findBrowser returns the path to Chrome or Chromium, preferring Chrome.
func findBrowser() (string, error) {
	candidates := browserCandidates()
	for _, c := range candidates {
		if path, err := exec.LookPath(c); err == nil {
			return path, nil
		}
	}
	return "", ErrNoBrowser
}
