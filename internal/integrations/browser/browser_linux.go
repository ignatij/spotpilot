//go:build linux

package browser

import (
	"context"
	"os/exec"
)

func browserCandidates() []string {
	return []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
	}
}

func openDefault(ctx context.Context, url string) error {
	return exec.CommandContext(ctx, "xdg-open", url).Start()
}
