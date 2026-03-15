package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"

	"github.com/ignatij/spotpilot/internal/app"
	"github.com/ignatij/spotpilot/internal/integrations/browser"
)

const spotifyLoginURL = "https://accounts.spotify.com/login"

// cdpLoginPerformer implements app.LoginPerformer using the Chrome DevTools Protocol.
// It launches Chrome with remote debugging enabled, navigates to the Spotify login
// page, and polls for the sp_dc session cookie that Spotify sets after a successful login.
type cdpLoginPerformer struct {
	pollInterval time.Duration
}

// NewCDPLoginPerformer returns a LoginPerformer backed by CDP.
func NewCDPLoginPerformer() app.LoginPerformer {
	return &cdpLoginPerformer{pollInterval: 2 * time.Second}
}

func (p *cdpLoginPerformer) PerformLogin(ctx context.Context) (*app.Session, error) {
	chromePath, err := browser.FindChromeBinary()
	if err != nil {
		return nil, fmt.Errorf("login requires Chrome or Chromium: %w", err)
	}

	// Launch Chrome in visible (non-headless) mode so the user can log in.
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-extensions", false),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	defer cancelBrowser()

	// Enable network events and navigate to the Spotify login page.
	if err := chromedp.Run(browserCtx,
		network.Enable(),
		chromedp.Navigate(spotifyLoginURL),
	); err != nil {
		return nil, fmt.Errorf("opening Spotify login page: %w", err)
	}

	fmt.Println("Waiting for Spotify login... (sign in in the browser window, then wait)")

	// Poll until the sp_dc cookie is set (login complete) or context expires.
	for {
		cookies, err := p.spotifyCookies(browserCtx)
		if err == nil && len(cookies) > 0 {
			return cookiesToSession(cookies), nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("login timed out or was cancelled: %w", ctx.Err())
		case <-time.After(p.pollInterval):
		}
	}
}

// spotifyCookies fetches all cookies from the browser and returns only the
// Spotify session cookies. Returns an error (causing a retry) if sp_dc is absent.
func (p *cdpLoginPerformer) spotifyCookies(ctx context.Context) ([]*network.Cookie, error) {
	var all []*network.Cookie
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		all, err = network.GetCookies().Do(ctx)
		return err
	}))
	if err != nil {
		return nil, err
	}

	var found []*network.Cookie
	for _, c := range all {
		if (c.Domain == ".spotify.com" || c.Domain == "spotify.com") && c.Name == "sp_dc" {
			found = append(found, c)
		}
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("sp_dc not yet present")
	}
	return found, nil
}

func cookiesToSession(cookies []*network.Cookie) *app.Session {
	sess := &app.Session{}
	for _, c := range cookies {
		sess.Cookies = append(sess.Cookies, app.Cookie{Name: c.Name, Value: c.Value})
	}
	return sess
}
