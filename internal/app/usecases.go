package app

import (
	"context"

	"github.com/ignatij/spotpilot/internal/domain"
)

// LoginInput is the input for the Login use case.
type LoginInput struct{}

// LoginResult is the result of the Login use case.
type LoginResult struct {
	AlreadyLoggedIn bool
}

// Login ensures a valid session exists. If one already exists it is a no-op.
// Otherwise it triggers the browser login flow and persists the session.
type Login struct {
	store    SessionStore
	launcher LoginPerformer
}

// NewLogin creates a Login use case.
func NewLogin(store SessionStore, launcher LoginPerformer) *Login {
	return &Login{store: store, launcher: launcher}
}

// Run executes the login use case.
func (u *Login) Run(ctx context.Context, _ LoginInput) (*LoginResult, error) {
	sess, err := u.store.Load(ctx)
	if err != nil {
		return nil, err
	}
	if sess != nil {
		return &LoginResult{AlreadyLoggedIn: true}, nil
	}

	newSess, err := u.launcher.PerformLogin(ctx)
	if err != nil {
		return nil, err
	}
	if err := u.store.Save(ctx, newSess); err != nil {
		return nil, err
	}
	return &LoginResult{AlreadyLoggedIn: false}, nil
}

// PlayInput is the input for the Play use case.
type PlayInput struct {
	Query string
}

// PlayResult is the result of the Play use case.
type PlayResult struct {
	Match    *domain.MatchResult
	Device   *domain.Device
	LoggedIn bool
}

// Play searches Spotify and plays the top result on the local device.
// It auto-triggers login if no valid session exists.
type Play struct {
	login    *Login
	spotify  SpotifyClient
	devices  DeviceDetector
	apps     AppLauncher
	browsers BrowserLauncher
}

// NewPlay creates a Play use case.
func NewPlay(login *Login, spotify SpotifyClient, devices DeviceDetector, apps AppLauncher, browsers BrowserLauncher) *Play {
	return &Play{login: login, spotify: spotify, devices: devices, apps: apps, browsers: browsers}
}

// Run executes the play use case.
func (u *Play) Run(ctx context.Context, in PlayInput) (*PlayResult, error) {
	if in.Query == "" {
		return nil, &appError{cat: catValidation, msg: "query is required"}
	}

	// Ensure login.
	loginRes, err := u.login.Run(ctx, LoginInput{})
	if err != nil {
		return nil, err
	}

	match, err := u.spotify.Search(ctx, in.Query)
	if err != nil {
		return nil, err
	}
	if match == nil {
		return nil, &appError{cat: catNotFound, msg: "no matching track, album, or artist found"}
	}

	device, err := u.resolveDevice(ctx)
	if err != nil {
		return nil, err
	}

	if err := u.spotify.Play(ctx, device.ID, match.PlaybackURI()); err != nil {
		return nil, err
	}

	return &PlayResult{
		Match:    match,
		Device:   device,
		LoggedIn: !loginRes.AlreadyLoggedIn,
	}, nil
}

// resolveDevice finds a local Spotify device, launching the desktop app and
// web player as needed.
func (u *Play) resolveDevice(ctx context.Context) (*domain.Device, error) {
	device, err := u.devices.WaitForLocalDevice(ctx)
	if err == nil {
		return device, nil
	}

	// Try launching the desktop app.
	_ = u.apps.LaunchSpotify(ctx)
	device, err = u.devices.WaitForLocalDevice(ctx)
	if err == nil {
		return device, nil
	}

	// Fall back to web player.
	_ = u.browsers.LaunchURL(ctx, "https://open.spotify.com")
	device, err = u.devices.WaitForLocalDevice(ctx)
	if err != nil {
		return nil, &appError{cat: catError, msg: "no local Spotify device found"}
	}
	return device, nil
}

// PlaybackInput is the input for simple playback control commands.
type PlaybackInput struct{}

// PlaybackResult is the result of a simple playback control command.
type PlaybackResult struct{}

// playbackUseCase is shared logic for pause/resume/next/previous.
type playbackUseCase struct {
	login   *Login
	spotify SpotifyClient
	devices DeviceDetector
	apps    AppLauncher
	browser BrowserLauncher
	action  func(ctx context.Context, deviceID string) error
}

func newPlaybackUseCase(login *Login, spotify SpotifyClient, devices DeviceDetector, apps AppLauncher, browser BrowserLauncher, action func(context.Context, string) error) *playbackUseCase {
	return &playbackUseCase{login: login, spotify: spotify, devices: devices, apps: apps, browser: browser, action: action}
}

func (u *playbackUseCase) Run(ctx context.Context, _ PlaybackInput) (*PlaybackResult, error) {
	if _, err := u.login.Run(ctx, LoginInput{}); err != nil {
		return nil, err
	}

	p := &Play{login: u.login, spotify: u.spotify, devices: u.devices, apps: u.apps, browsers: u.browser}
	device, err := p.resolveDevice(ctx)
	if err != nil {
		return nil, err
	}

	if err := u.action(ctx, device.ID); err != nil {
		return nil, err
	}
	return &PlaybackResult{}, nil
}

// Pause use case.
type Pause struct{ inner *playbackUseCase }

func NewPause(login *Login, spotify SpotifyClient, devices DeviceDetector, apps AppLauncher, browser BrowserLauncher) *Pause {
	return &Pause{inner: newPlaybackUseCase(login, spotify, devices, apps, browser, spotify.Pause)}
}
func (u *Pause) Run(ctx context.Context, in PlaybackInput) (*PlaybackResult, error) {
	return u.inner.Run(ctx, in)
}

// Resume use case.
type Resume struct{ inner *playbackUseCase }

func NewResume(login *Login, spotify SpotifyClient, devices DeviceDetector, apps AppLauncher, browser BrowserLauncher) *Resume {
	return &Resume{inner: newPlaybackUseCase(login, spotify, devices, apps, browser, spotify.Resume)}
}
func (u *Resume) Run(ctx context.Context, in PlaybackInput) (*PlaybackResult, error) {
	return u.inner.Run(ctx, in)
}

// Next use case.
type Next struct{ inner *playbackUseCase }

func NewNext(login *Login, spotify SpotifyClient, devices DeviceDetector, apps AppLauncher, browser BrowserLauncher) *Next {
	return &Next{inner: newPlaybackUseCase(login, spotify, devices, apps, browser, spotify.Next)}
}
func (u *Next) Run(ctx context.Context, in PlaybackInput) (*PlaybackResult, error) {
	return u.inner.Run(ctx, in)
}

// Previous use case.
type Previous struct{ inner *playbackUseCase }

func NewPrevious(login *Login, spotify SpotifyClient, devices DeviceDetector, apps AppLauncher, browser BrowserLauncher) *Previous {
	return &Previous{inner: newPlaybackUseCase(login, spotify, devices, apps, browser, spotify.Previous)}
}
func (u *Previous) Run(ctx context.Context, in PlaybackInput) (*PlaybackResult, error) {
	return u.inner.Run(ctx, in)
}

// StatusInput is the input for the Status use case.
type StatusInput struct{}

// StatusResult is the result of the Status use case.
type StatusResult struct {
	LoggedIn bool
	Playback *domain.CurrentPlayback
}

// Status reports current playback state without any side effects.
type Status struct {
	store   SessionStore
	spotify SpotifyClient
}

// NewStatus creates a Status use case.
func NewStatus(store SessionStore, spotify SpotifyClient) *Status {
	return &Status{store: store, spotify: spotify}
}

// Run executes the status use case.
func (u *Status) Run(ctx context.Context, _ StatusInput) (*StatusResult, error) {
	sess, err := u.store.Load(ctx)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return &StatusResult{LoggedIn: false}, nil
	}

	pb, err := u.spotify.CurrentPlayback(ctx)
	if err != nil {
		return nil, err
	}
	return &StatusResult{LoggedIn: true, Playback: pb}, nil
}
