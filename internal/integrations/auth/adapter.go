package auth

import (
	"context"
	"fmt"

	"github.com/ignatij/spotpilot/internal/app"
)

// AppStoreAdapter adapts FileStore to the app.SessionStore port.
type AppStoreAdapter struct {
	inner *FileStore
}

// NewAppStoreAdapter wraps a FileStore to satisfy app.SessionStore.
func NewAppStoreAdapter(inner *FileStore) *AppStoreAdapter {
	return &AppStoreAdapter{inner: inner}
}

func (a *AppStoreAdapter) Load(ctx context.Context) (*app.Session, error) {
	sess, err := a.inner.Load(ctx)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, nil
	}
	out := &app.Session{}
	for _, c := range sess.Cookies {
		out.Cookies = append(out.Cookies, app.Cookie{Name: c.Name, Value: c.Value})
	}
	return out, nil
}

func (a *AppStoreAdapter) Save(ctx context.Context, s *app.Session) error {
	inner := &Session{}
	for _, c := range s.Cookies {
		inner.Cookies = append(inner.Cookies, Cookie{Name: c.Name, Value: c.Value})
	}
	return a.inner.Save(ctx, inner)
}

func (a *AppStoreAdapter) Clear(ctx context.Context) error {
	return a.inner.Clear(ctx)
}

// LoginPerformer implements the app.LoginPerformer port.
// It opens Chrome/Chromium, waits for the user to log in to Spotify,
// then imports the browser cookies.
//
// NOTE: The cookie import mechanism (reading the Chrome cookie database) is
// complex and platform-specific. This implementation provides the scaffolding;
// the real cookie extraction is a separate integration concern.
type LoginPerformer struct {
	store   *FileStore
	browser browserOpener
}

type browserOpener interface {
	LaunchLogin(ctx context.Context, url string) error
}

// NewLoginPerformer creates a LoginPerformer.
func NewLoginPerformer(store *FileStore, browser browserOpener) *LoginPerformer {
	return &LoginPerformer{store: store, browser: browser}
}

// PerformLogin opens the Spotify login page in Chrome/Chromium, waits for the
// user to authenticate, and returns the imported session.
func (l *LoginPerformer) PerformLogin(ctx context.Context) (*app.Session, error) {
	const spotifyLoginURL = "https://accounts.spotify.com/login"

	if err := l.browser.LaunchLogin(ctx, spotifyLoginURL); err != nil {
		return nil, fmt.Errorf("launching browser for login: %w", err)
	}

	// TODO: implement cookie import from Chrome/Chromium.
	// For now, block until ctx is done so callers get a clear signal.
	<-ctx.Done()
	return nil, fmt.Errorf("login not yet implemented: %w", ctx.Err())
}
