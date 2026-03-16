//go:build darwin

package browser

import (
	"context"
	"os/exec"
	"path/filepath"
)

func browserCandidates() []string {
	return []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"google-chrome",
		"chromium",
		"chromium-browser",
	}
}

func userDataDirCandidates(homeDir string) []string {
	return []string{
		filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome"),
		filepath.Join(homeDir, "Library", "Application Support", "Chromium"),
	}
}

func openDefault(ctx context.Context, url string) error {
	return exec.CommandContext(ctx, "open", url).Start()
}
