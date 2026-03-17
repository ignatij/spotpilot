package app

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	if hasPlaybackSession(sess) {
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
	apps     Launcher
	browsers BrowserLauncher
}

// NewPlay creates a Play use case.
func NewPlay(login *Login, spotify SpotifyClient, devices DeviceDetector, apps Launcher, browsers BrowserLauncher) *Play {
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
	match = u.refineMatchForPlay(ctx, in.Query, match)
	match = u.ensurePlayableMatch(ctx, in.Query, match)
	if match == nil {
		return nil, &appError{cat: catNotFound, msg: "no matching track, album, or artist found"}
	}

	device, err := u.resolveDevice(ctx)
	if err != nil {
		return nil, err
	}

	playURI := strings.TrimSpace(match.PlaybackURI())
	if playURI == "" {
		return nil, &appError{cat: catNotFound, msg: "no playable Spotify URI found for query"}
	}

	if err := u.spotify.Play(ctx, device.ID, playURI); err != nil {
		if isBlockedPlaybackError(err) {
			uri := playURI
			if uri != "" {
				if launchErr := u.browsers.LaunchURL(ctx, uri); launchErr == nil {
					match = u.enrichMatchFromCurrentPlayback(ctx, match)
					return &PlayResult{
						Match:    match,
						Device:   device,
						LoggedIn: !loginRes.AlreadyLoggedIn,
					}, nil
				}
				if webURL := spotifyURIToWebURL(uri); webURL != "" {
					if launchErr := u.browsers.LaunchURL(ctx, webURL); launchErr == nil {
						match = u.enrichMatchFromCurrentPlayback(ctx, match)
						return &PlayResult{
							Match:    match,
							Device:   device,
							LoggedIn: !loginRes.AlreadyLoggedIn,
						}, nil
					}
				}
			}
		}
		return nil, err
	}
	if err := u.verifyRequestedPlayback(ctx, match); err != nil {
		return nil, err
	}

	match = u.enrichMatchFromCurrentPlayback(ctx, match)

	return &PlayResult{
		Match:    match,
		Device:   device,
		LoggedIn: !loginRes.AlreadyLoggedIn,
	}, nil
}

func (u *Play) verifyRequestedPlayback(ctx context.Context, match *domain.MatchResult) error {
	if match == nil || match.Type != domain.MatchTypeTrack || match.Track == nil {
		return nil
	}
	expected := strings.TrimSpace(match.Track.URI)
	if expected == "" {
		return nil
	}

	const maxAttempts = 4
	const verificationWait = 250 * time.Millisecond

	observed := ""
	for attempt := 0; attempt < maxAttempts; attempt++ {
		pb, err := u.spotify.CurrentPlayback(ctx)
		if err == nil && pb != nil && pb.Track != nil {
			observed = strings.TrimSpace(pb.Track.URI)
			if observed == expected {
				return nil
			}
		}
		if attempt == maxAttempts-1 {
			break
		}
		timer := time.NewTimer(verificationWait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	if observed == "" {
		return &appError{cat: catError, msg: "playback could not be verified for requested track"}
	}
	return &appError{cat: catError, msg: "playback stayed on a different track", err: fmt.Errorf("expected %s got %s", expected, observed)}
}

func (u *Play) ensurePlayableMatch(ctx context.Context, query string, match *domain.MatchResult) *domain.MatchResult {
	if hasPlayableURI(match) {
		return match
	}

	queries := []string{
		strings.TrimSpace(query) + " track",
		"track:" + strings.TrimSpace(query),
		strings.TrimSpace(query) + " song",
	}
	for _, q := range queries {
		candidate, err := u.spotify.Search(ctx, q)
		if err != nil || candidate == nil {
			continue
		}
		if hasPlayableURI(candidate) {
			return candidate
		}
	}

	if match == nil {
		return nil
	}
	return match
}

func hasPlayableURI(match *domain.MatchResult) bool {
	if match == nil {
		return false
	}
	return strings.TrimSpace(match.PlaybackURI()) != ""
}

func (u *Play) enrichMatchFromCurrentPlayback(ctx context.Context, match *domain.MatchResult) *domain.MatchResult {
	if match == nil {
		return nil
	}
	pb, err := u.spotify.CurrentPlayback(ctx)
	if err != nil || pb == nil || pb.Track == nil {
		return match
	}

	title := strings.TrimSpace(pb.Track.Title)
	artist := strings.TrimSpace(pb.Track.Artist)
	album := strings.TrimSpace(pb.Track.Album)
	uri := strings.TrimSpace(pb.Track.URI)
	if title == "" && artist == "" {
		return match
	}

	if match.Type == domain.MatchTypeTrack {
		if match.Track == nil {
			match.Track = &domain.Track{}
		}
		if strings.TrimSpace(match.Track.URI) == "" {
			match.Track.URI = uri
		}
		if strings.TrimSpace(match.Track.Title) == "" {
			match.Track.Title = title
		}
		if strings.TrimSpace(match.Track.Artist) == "" {
			match.Track.Artist = artist
		}
		if strings.TrimSpace(match.Track.Album) == "" {
			match.Track.Album = album
		}
		return match
	}

	if (match.Type == domain.MatchTypeAlbum && (match.Album == nil || strings.TrimSpace(match.Album.Name) == "")) ||
		(match.Type == domain.MatchTypeArtist && (match.Artist == nil || strings.TrimSpace(match.Artist.Name) == "")) {
		return &domain.MatchResult{
			Type: domain.MatchTypeTrack,
			Track: &domain.Track{
				URI:    firstNonEmpty(uri, match.PlaybackURI()),
				Title:  title,
				Artist: artist,
				Album:  album,
			},
		}
	}

	return match
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (u *Play) refineMatchForPlay(ctx context.Context, query string, match *domain.MatchResult) *domain.MatchResult {
	if !needsTrackRefinement(query, match) {
		return match
	}
	queries := []string{
		strings.TrimSpace(query) + " track",
		"track:" + strings.TrimSpace(query),
	}
	for _, q := range queries {
		refined, err := u.spotify.Search(ctx, q)
		if err != nil || refined == nil {
			continue
		}
		if refined.Type == domain.MatchTypeTrack {
			return refined
		}
	}
	return match
}

func needsTrackRefinement(query string, match *domain.MatchResult) bool {
	if match == nil {
		return false
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if strings.Contains(q, "artist") {
		return false
	}
	switch match.Type {
	case domain.MatchTypeArtist:
		if match.Artist == nil {
			return true
		}
		name := strings.TrimSpace(match.Artist.Name)
		uri := strings.TrimSpace(match.Artist.URI)
		return name == "" || uri == "" || !strings.Contains(q, "artist")
	case domain.MatchTypeAlbum:
		if strings.Contains(q, "album") {
			return false
		}
		if match.Album == nil {
			return true
		}
		return strings.TrimSpace(match.Album.Name) == "" || strings.TrimSpace(match.Album.URI) == ""
	default:
		return false
	}
}

func isBlockedPlaybackError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "status 410") && strings.Contains(text, "status 429")
}

func spotifyURIToWebURL(uri string) string {
	parts := strings.Split(strings.TrimSpace(uri), ":")
	if len(parts) < 3 || !strings.EqualFold(parts[0], "spotify") {
		return ""
	}
	entity := strings.TrimSpace(parts[1])
	id := strings.TrimSpace(parts[2])
	if entity == "" || id == "" {
		return ""
	}
	return fmt.Sprintf("https://open.spotify.com/%s/%s", entity, id)
}

// resolveDevice finds a local Spotify device, launching the desktop app and
// web player as needed.
func (u *Play) resolveDevice(ctx context.Context) (*domain.Device, error) {
	device, err := u.devices.WaitForLocalDevice(ctx)
	if err == nil {
		return device, nil
	}
	initialWaitErr := err

	// Try launching the desktop app.
	desktopLaunchErr := u.apps.LaunchSpotify(ctx)
	device, err = u.devices.WaitForLocalDevice(ctx)
	if err == nil {
		return device, nil
	}
	desktopWaitErr := err

	// Fall back to web player.
	webLaunchErr := u.browsers.LaunchURL(ctx, "https://open.spotify.com")
	device, err = u.devices.WaitForLocalDevice(ctx)
	if err != nil {
		return nil, &appError{cat: catError, msg: noLocalDeviceMessage(initialWaitErr, desktopLaunchErr, desktopWaitErr, webLaunchErr, err)}
	}
	return device, nil
}

func noLocalDeviceMessage(initialWaitErr, desktopLaunchErr, desktopWaitErr, webLaunchErr, webWaitErr error) string {
	parts := []string{"no local Spotify device found"}
	if initialWaitErr != nil {
		parts = append(parts, fmt.Sprintf("initial_scan=%v", initialWaitErr))
	}
	if desktopLaunchErr != nil || desktopWaitErr != nil {
		desktop := make([]string, 0, 2)
		if desktopLaunchErr != nil {
			desktop = append(desktop, fmt.Sprintf("launch=%v", desktopLaunchErr))
		}
		if desktopWaitErr != nil {
			desktop = append(desktop, fmt.Sprintf("wait=%v", desktopWaitErr))
		}
		parts = append(parts, "desktop_attempt={"+strings.Join(desktop, ", ")+"}")
	}
	if webLaunchErr != nil || webWaitErr != nil {
		web := make([]string, 0, 2)
		if webLaunchErr != nil {
			web = append(web, fmt.Sprintf("launch=%v", webLaunchErr))
		}
		if webWaitErr != nil {
			web = append(web, fmt.Sprintf("wait=%v", webWaitErr))
		}
		parts = append(parts, "web_attempt={"+strings.Join(web, ", ")+"}")
	}
	return strings.Join(parts, "; ")
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
	apps    Launcher
	browser BrowserLauncher
	action  func(ctx context.Context, deviceID string) error
}

func newPlaybackUseCase(login *Login, spotify SpotifyClient, devices DeviceDetector, apps Launcher, browser BrowserLauncher, action func(context.Context, string) error) *playbackUseCase {
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

func NewPause(login *Login, spotify SpotifyClient, devices DeviceDetector, apps Launcher, browser BrowserLauncher) *Pause {
	return &Pause{inner: newPlaybackUseCase(login, spotify, devices, apps, browser, spotify.Pause)}
}
func (u *Pause) Run(ctx context.Context, in PlaybackInput) (*PlaybackResult, error) {
	return u.inner.Run(ctx, in)
}

// Resume use case.
type Resume struct{ inner *playbackUseCase }

func NewResume(login *Login, spotify SpotifyClient, devices DeviceDetector, apps Launcher, browser BrowserLauncher) *Resume {
	return &Resume{inner: newPlaybackUseCase(login, spotify, devices, apps, browser, spotify.Resume)}
}
func (u *Resume) Run(ctx context.Context, in PlaybackInput) (*PlaybackResult, error) {
	return u.inner.Run(ctx, in)
}

// Next use case.
type Next struct{ inner *playbackUseCase }

func NewNext(login *Login, spotify SpotifyClient, devices DeviceDetector, apps Launcher, browser BrowserLauncher) *Next {
	return &Next{inner: newPlaybackUseCase(login, spotify, devices, apps, browser, spotify.Next)}
}
func (u *Next) Run(ctx context.Context, in PlaybackInput) (*PlaybackResult, error) {
	return u.inner.Run(ctx, in)
}

// Previous use case.
type Previous struct{ inner *playbackUseCase }

func NewPrevious(login *Login, spotify SpotifyClient, devices DeviceDetector, apps Launcher, browser BrowserLauncher) *Previous {
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
	if !hasSessionCookie(sess, "sp_dc") {
		return &StatusResult{LoggedIn: false}, nil
	}

	pb, err := u.spotify.CurrentPlayback(ctx)
	if err != nil {
		return nil, err
	}
	return &StatusResult{LoggedIn: true, Playback: pb}, nil
}

func hasSessionCookie(sess *Session, name string) bool {
	if sess == nil {
		return false
	}
	for _, cookie := range sess.Cookies {
		if cookie.Name == name && cookie.Value != "" {
			return true
		}
	}
	return false
}

func hasPlaybackSession(sess *Session) bool {
	return hasSessionCookie(sess, "sp_dc") && hasSessionCookie(sess, "sp_t")
}
