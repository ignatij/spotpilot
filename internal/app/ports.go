package app

import (
	"context"
	"time"

	"github.com/ignatij/spotpilot/internal/domain"
)

// SpotifyClient is the consumer-owned port for Spotify Web API interactions.
type SpotifyClient interface {
	Search(ctx context.Context, query string) (*domain.MatchResult, error)
	Play(ctx context.Context, deviceID string, uri string) error
	Pause(ctx context.Context, deviceID string) error
	Resume(ctx context.Context, deviceID string) error
	Next(ctx context.Context, deviceID string) error
	Previous(ctx context.Context, deviceID string) error
	CurrentPlayback(ctx context.Context) (*domain.CurrentPlayback, error)
	ListDevices(ctx context.Context) ([]domain.Device, error)
}

// SessionStore is the consumer-owned port for session/credential persistence.
type SessionStore interface {
	// Load returns (nil, nil) when no session exists yet.
	Load(ctx context.Context) (*Session, error)
	Save(ctx context.Context, s *Session) error
	Clear(ctx context.Context) error
}

// Session holds a persisted Spotify session.
type Session struct {
	// Cookies contains the raw browser session cookies.
	Cookies     []Cookie
	AccessToken string
	TokenExpiry time.Time
}

// Cookie is a browser session cookie.
type Cookie struct {
	Name  string
	Value string
}

// BrowserLauncher is the consumer-owned port for launching browsers.
type BrowserLauncher interface {
	// LaunchLogin opens the given URL in Chrome or Chromium for session import.
	// Returns ErrNoBrowser if neither Chrome nor Chromium is installed.
	LaunchLogin(ctx context.Context, url string) error
	// LaunchURL opens the given URL in the system default browser.
	LaunchURL(ctx context.Context, url string) error
}

// DeviceDetector is the consumer-owned port for detecting local Spotify devices.
type DeviceDetector interface {
	// WaitForLocalDevice waits up to timeout for a local Spotify device and returns it.
	WaitForLocalDevice(ctx context.Context) (*domain.Device, error)
}

// Launcher is the consumer-owned port for launching local applications.
type Launcher interface {
	// LaunchSpotify starts the local Spotify desktop application.
	LaunchSpotify(ctx context.Context) error
}

// LoginPerformer is the consumer-owned port for executing the OAuth/cookie login flow.
type LoginPerformer interface {
	// PerformLogin opens the browser, waits for user login, imports cookies,
	// and returns a Session. Blocks until login completes or ctx is cancelled.
	PerformLogin(ctx context.Context) (*Session, error)
}
