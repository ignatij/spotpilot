//go:build linux

package browser

import (
	"context"
	"os/exec"
	"path/filepath"
)

func browserCandidates() []string {
	return []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
	}
}

func userDataDirCandidates(homeDir string) []string {
	return []string{
		filepath.Join(homeDir, ".config", "google-chrome"),
		filepath.Join(homeDir, ".config", "chromium"),
	}
}

func openDefault(ctx context.Context, url string) error {
	return exec.CommandContext(ctx, "xdg-open", url).Start()
}
