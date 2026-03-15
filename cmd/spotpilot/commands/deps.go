package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/ignatij/spotpilot/internal/app"
	"github.com/ignatij/spotpilot/internal/config"
	"github.com/ignatij/spotpilot/internal/integrations/auth"
	"github.com/ignatij/spotpilot/internal/integrations/browser"
	"github.com/ignatij/spotpilot/internal/integrations/spotify"
)

// deps holds all resolved runtime dependencies for command execution.
type deps struct {
	store          app.SessionStore
	loginPerformer app.LoginPerformer
	spotify        app.SpotifyClient
	devices        app.DeviceDetector
	appLauncher    app.AppLauncher
	browser        app.BrowserLauncher
}

// buildDeps constructs and wires all runtime dependencies.
// Heavy initialization is deferred to here so startup stays cheap.
func buildDeps(cfg config.Config) (*deps, error) {
	fileStore, err := auth.NewFileStore("")
	if err != nil {
		return nil, fmt.Errorf("initializing session store: %w", err)
	}

	b := browser.New()

	store := auth.NewAppStoreAdapter(fileStore)
	loginPerformer := auth.NewCDPLoginPerformer()

	sess, _ := store.Load(context.Background())
	var token string
	if sess != nil && len(sess.Cookies) > 0 {
		// Extract sp_dc cookie as Bearer-equivalent token.
		for _, c := range sess.Cookies {
			if c.Name == "sp_dc" {
				token = c.Value
				break
			}
		}
	}

	spotifyClient := spotify.New(func() (string, error) {
		if token == "" {
			return "", fmt.Errorf("not authenticated")
		}
		return token, nil
	})

	hostname, _ := os.Hostname()
	deviceDetector := spotify.NewDeviceDetector(spotifyClient, hostname)
	appLauncher := spotify.NewAppLauncher()

	return &deps{
		store:          store,
		loginPerformer: loginPerformer,
		spotify:        spotifyClient,
		devices:        deviceDetector,
		appLauncher:    appLauncher,
		browser:        b,
	}, nil
}
