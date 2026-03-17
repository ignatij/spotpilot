package commands

import (
	"context"
	"fmt"
	"net/http"
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
	appLauncher    app.Launcher
	browser        app.BrowserLauncher
}

// buildDeps constructs and wires all runtime dependencies.
// Heavy initialization is deferred to here so startup stays cheap.
func buildDeps(_ config.Config) (*deps, error) {
	cs, err := auth.NewAutoStore()
	if err != nil {
		return nil, fmt.Errorf("initializing session store: %w", err)
	}

	b := browser.New()

	store := auth.NewAppStoreAdapter(cs)
	loginPerformer := auth.NewCDPLoginPerformer()

	cookieSource := spotify.CookieSourceFunc(func(ctx context.Context) ([]*http.Cookie, error) {
		sess, err := cs.Load(ctx)
		if err != nil {
			return nil, err
		}
		if sess == nil {
			return nil, fmt.Errorf("not authenticated — run 'spotpilot login' first")
		}

		cookies := make([]*http.Cookie, 0, len(sess.Cookies))
		for _, cookie := range sess.Cookies {
			if cookie.Value == "" {
				continue
			}
			cookies = append(cookies, &http.Cookie{
				Name:     cookie.Name,
				Value:    cookie.Value,
				Domain:   ".spotify.com",
				Path:     "/",
				Secure:   true,
				HttpOnly: true,
			})
		}
		if len(cookies) == 0 {
			return nil, fmt.Errorf("no Spotify session cookies found — run 'spotpilot login' again")
		}
		return cookies, nil
	})

	cookieTokenProvider := spotify.CookieTokenProvider{Source: cookieSource}

	webClient := spotify.New(cookieTokenProvider.Token)
	connectClient, err := spotify.NewConnectClient(cookieSource, webClient)
	if err != nil {
		return nil, fmt.Errorf("initializing connect playback client: %w", err)
	}
	spotifyClient := spotify.NewHybridClient(webClient, connectClient)

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
