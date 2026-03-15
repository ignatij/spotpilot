//go:build darwin

package browser

import (
	"context"
	"os/exec"
)

func browserCandidates() []string {
	return []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"google-chrome",
		"chromium",
		"chromium-browser",
	}
}

func openDefault(ctx context.Context, url string) error {
	return exec.CommandContext(ctx, "open", url).Start()
}
