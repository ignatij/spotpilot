package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ignatij/spotpilot/internal/app"
	"github.com/ignatij/spotpilot/internal/domain"
)

// --- fakes ---

type fakeStore struct {
	session *app.Session
	saved   *app.Session
	cleared bool
}

func (f *fakeStore) Load(_ context.Context) (*app.Session, error) { return f.session, nil }
func (f *fakeStore) Save(_ context.Context, s *app.Session) error { f.saved = s; return nil }
func (f *fakeStore) Clear(_ context.Context) error                { f.cleared = true; return nil }

type fakeLoginPerformer struct {
	session *app.Session
	err     error
}

func (f *fakeLoginPerformer) PerformLogin(_ context.Context) (*app.Session, error) {
	return f.session, f.err
}

type fakeSpotify struct {
	searchResult  *domain.MatchResult
	searchResults map[string]*domain.MatchResult
	devices       []domain.Device
	playback      *domain.CurrentPlayback
	playbacks     []*domain.CurrentPlayback
	playErr       error
	playedURI     string
	playCalls     int
}

func (f *fakeSpotify) Search(_ context.Context, query string) (*domain.MatchResult, error) {
	if f.searchResults != nil {
		if result, ok := f.searchResults[query]; ok {
			return result, nil
		}
	}
	return f.searchResult, nil
}
func (f *fakeSpotify) Play(_ context.Context, _ string, uri string) error {
	f.playCalls++
	f.playedURI = uri
	return f.playErr
}
func (f *fakeSpotify) Pause(_ context.Context, _ string) error    { return nil }
func (f *fakeSpotify) Resume(_ context.Context, _ string) error   { return nil }
func (f *fakeSpotify) Next(_ context.Context, _ string) error     { return nil }
func (f *fakeSpotify) Previous(_ context.Context, _ string) error { return nil }
func (f *fakeSpotify) CurrentPlayback(_ context.Context) (*domain.CurrentPlayback, error) {
	if len(f.playbacks) > 0 {
		pb := f.playbacks[0]
		if len(f.playbacks) > 1 {
			f.playbacks = f.playbacks[1:]
		}
		return pb, nil
	}
	return f.playback, nil
}
func (f *fakeSpotify) ListDevices(_ context.Context) ([]domain.Device, error) {
	return f.devices, nil
}

type fakeDeviceDetector struct {
	device *domain.Device
	err    error
}

func (f *fakeDeviceDetector) WaitForLocalDevice(_ context.Context) (*domain.Device, error) {
	return f.device, f.err
}

type fakeAppLauncher struct{}

func (f *fakeAppLauncher) LaunchSpotify(_ context.Context) error { return nil }

type fakeBrowserLauncher struct{}

func (f *fakeBrowserLauncher) LaunchLogin(_ context.Context, _ string) error { return nil }
func (f *fakeBrowserLauncher) LaunchURL(_ context.Context, _ string) error   { return nil }

// --- tests ---

func TestLogin_AlreadyLoggedIn(t *testing.T) {
	store := &fakeStore{session: &app.Session{Cookies: []app.Cookie{{Name: "sp_dc", Value: "tok"}, {Name: "sp_t", Value: "device"}}}}
	performer := &fakeLoginPerformer{}
	uc := app.NewLogin(store, performer)

	res, err := uc.Run(context.Background(), app.LoginInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.AlreadyLoggedIn {
		t.Error("expected AlreadyLoggedIn=true when session exists")
	}
}

func TestLogin_NoSession_CallsPerformer(t *testing.T) {
	store := &fakeStore{session: nil}
	newSess := &app.Session{Cookies: []app.Cookie{{Name: "sp_dc", Value: "tok"}}}
	performer := &fakeLoginPerformer{session: newSess}
	uc := app.NewLogin(store, performer)

	res, err := uc.Run(context.Background(), app.LoginInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.AlreadyLoggedIn {
		t.Error("expected AlreadyLoggedIn=false for new login")
	}
	if store.saved == nil {
		t.Error("expected session to be saved")
	}
}

func TestPlay_EmptyQuery_Resumes(t *testing.T) {
	store := &fakeStore{session: &app.Session{}}
	performer := &fakeLoginPerformer{}
	loginUC := app.NewLogin(store, performer)
	spotifyClient := &fakeSpotify{}
	devices := &fakeDeviceDetector{device: &domain.Device{ID: "d1", Type: "Computer"}}

	uc := app.NewPlay(loginUC, spotifyClient, devices, &fakeAppLauncher{}, &fakeBrowserLauncher{})
	res, err := uc.Run(context.Background(), app.PlayInput{Query: ""})
	if err != nil {
		t.Fatalf("unexpected error on empty query resume: %v", err)
	}
	if res == nil {
		t.Fatal("expected result")
		return
	}
	if res.Match != nil {
		t.Errorf("expected nil match for resume, got %+v", res.Match)
	}
}

func TestPlay_NotFound(t *testing.T) {
	store := &fakeStore{session: &app.Session{}}
	performer := &fakeLoginPerformer{}
	loginUC := app.NewLogin(store, performer)
	spotifyClient := &fakeSpotify{searchResult: nil} // no match
	devices := &fakeDeviceDetector{device: &domain.Device{ID: "d1", Type: "Computer"}}

	uc := app.NewPlay(loginUC, spotifyClient, devices, &fakeAppLauncher{}, &fakeBrowserLauncher{})
	_, err := uc.Run(context.Background(), app.PlayInput{Query: "nonexistent"})
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestPlay_Success(t *testing.T) {
	store := &fakeStore{session: &app.Session{}}
	performer := &fakeLoginPerformer{}
	loginUC := app.NewLogin(store, performer)
	spotifyClient := &fakeSpotify{
		searchResult: &domain.MatchResult{
			Type:  domain.MatchTypeTrack,
			Track: &domain.Track{URI: "spotify:track:1", Title: "Master of Puppets", Artist: "Metallica"},
		},
		playback: &domain.CurrentPlayback{
			State: domain.PlaybackStatePlaying,
			Track: &domain.Track{URI: "spotify:track:1", Title: "Master of Puppets", Artist: "Metallica"},
		},
	}
	devices := &fakeDeviceDetector{device: &domain.Device{ID: "d1", Type: "Computer"}}

	uc := app.NewPlay(loginUC, spotifyClient, devices, &fakeAppLauncher{}, &fakeBrowserLauncher{})
	res, err := uc.Run(context.Background(), app.PlayInput{Query: "Master of Puppets"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Match.Track.Title != "Master of Puppets" {
		t.Errorf("unexpected title: %q", res.Match.Track.Title)
	}
}

func TestPlayVerificationMismatch(t *testing.T) {
	store := &fakeStore{session: &app.Session{}}
	performer := &fakeLoginPerformer{}
	loginUC := app.NewLogin(store, performer)
	spotifyClient := &fakeSpotify{
		searchResult: &domain.MatchResult{
			Type:  domain.MatchTypeTrack,
			Track: &domain.Track{URI: "spotify:track:battery", Title: "Battery", Artist: "Metallica"},
		},
		playbacks: []*domain.CurrentPlayback{
			{State: domain.PlaybackStatePlaying, Track: &domain.Track{URI: "spotify:track:enter-sandman", Title: "Enter Sandman", Artist: "Metallica"}},
			{State: domain.PlaybackStatePlaying, Track: &domain.Track{URI: "spotify:track:enter-sandman", Title: "Enter Sandman", Artist: "Metallica"}},
			{State: domain.PlaybackStatePlaying, Track: &domain.Track{URI: "spotify:track:enter-sandman", Title: "Enter Sandman", Artist: "Metallica"}},
			{State: domain.PlaybackStatePlaying, Track: &domain.Track{URI: "spotify:track:enter-sandman", Title: "Enter Sandman", Artist: "Metallica"}},
		},
	}
	devices := &fakeDeviceDetector{device: &domain.Device{ID: "d1", Type: "Computer"}}

	uc := app.NewPlay(loginUC, spotifyClient, devices, &fakeAppLauncher{}, &fakeBrowserLauncher{})
	_, err := uc.Run(context.Background(), app.PlayInput{Query: "Battery"})
	if err == nil {
		t.Fatal("expected error when playback stays on a different track")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "different track") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatus_NotLoggedIn(t *testing.T) {
	store := &fakeStore{session: nil}
	spotifyClient := &fakeSpotify{}
	uc := app.NewStatus(store, spotifyClient)

	res, err := uc.Run(context.Background(), app.StatusInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.LoggedIn {
		t.Error("expected LoggedIn=false when no session")
	}
}

func TestStatus_Idle(t *testing.T) {
	store := &fakeStore{session: &app.Session{Cookies: []app.Cookie{{Name: "sp_dc", Value: "tok"}}}}
	spotifyClient := &fakeSpotify{
		playback: &domain.CurrentPlayback{State: domain.PlaybackStateIdle},
	}
	uc := app.NewStatus(store, spotifyClient)

	res, err := uc.Run(context.Background(), app.StatusInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.LoggedIn {
		t.Error("expected LoggedIn=true")
	}
	if res.Playback.State != domain.PlaybackStateIdle {
		t.Errorf("expected idle, got %q", res.Playback.State)
	}
}

// helpers shared by playback control tests
func loggedInStore() *fakeStore {
	return &fakeStore{session: &app.Session{Cookies: []app.Cookie{{Name: "sp_dc", Value: "tok"}, {Name: "sp_t", Value: "device"}}}}
}
func localDevice() *fakeDeviceDetector {
	return &fakeDeviceDetector{device: &domain.Device{ID: "d1", Type: "Computer"}}
}
func noDevice() *fakeDeviceDetector {
	return &fakeDeviceDetector{err: errors.New("no device")}
}

func TestPause_Success(t *testing.T) {
	uc := app.NewPause(app.NewLogin(loggedInStore(), &fakeLoginPerformer{}), &fakeSpotify{}, localDevice(), &fakeAppLauncher{}, &fakeBrowserLauncher{})
	if _, err := uc.Run(context.Background(), app.PlaybackInput{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPause_NoDevice(t *testing.T) {
	uc := app.NewPause(app.NewLogin(loggedInStore(), &fakeLoginPerformer{}), &fakeSpotify{}, noDevice(), &fakeAppLauncher{}, &fakeBrowserLauncher{})
	if _, err := uc.Run(context.Background(), app.PlaybackInput{}); err == nil {
		t.Fatal("expected error when no device found")
	}
}

func TestResume_Success(t *testing.T) {
	uc := app.NewResume(app.NewLogin(loggedInStore(), &fakeLoginPerformer{}), &fakeSpotify{}, localDevice(), &fakeAppLauncher{}, &fakeBrowserLauncher{})
	if _, err := uc.Run(context.Background(), app.PlaybackInput{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResume_NoDevice(t *testing.T) {
	uc := app.NewResume(app.NewLogin(loggedInStore(), &fakeLoginPerformer{}), &fakeSpotify{}, noDevice(), &fakeAppLauncher{}, &fakeBrowserLauncher{})
	if _, err := uc.Run(context.Background(), app.PlaybackInput{}); err == nil {
		t.Fatal("expected error when no device found")
	}
}

func TestNext_Success(t *testing.T) {
	uc := app.NewNext(app.NewLogin(loggedInStore(), &fakeLoginPerformer{}), &fakeSpotify{}, localDevice(), &fakeAppLauncher{}, &fakeBrowserLauncher{})
	if _, err := uc.Run(context.Background(), app.PlaybackInput{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNext_NoDevice(t *testing.T) {
	uc := app.NewNext(app.NewLogin(loggedInStore(), &fakeLoginPerformer{}), &fakeSpotify{}, noDevice(), &fakeAppLauncher{}, &fakeBrowserLauncher{})
	if _, err := uc.Run(context.Background(), app.PlaybackInput{}); err == nil {
		t.Fatal("expected error when no device found")
	}
}

func TestPrevious_Success(t *testing.T) {
	uc := app.NewPrevious(app.NewLogin(loggedInStore(), &fakeLoginPerformer{}), &fakeSpotify{}, localDevice(), &fakeAppLauncher{}, &fakeBrowserLauncher{})
	if _, err := uc.Run(context.Background(), app.PlaybackInput{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrevious_NoDevice(t *testing.T) {
	uc := app.NewPrevious(app.NewLogin(loggedInStore(), &fakeLoginPerformer{}), &fakeSpotify{}, noDevice(), &fakeAppLauncher{}, &fakeBrowserLauncher{})
	if _, err := uc.Run(context.Background(), app.PlaybackInput{}); err == nil {
		t.Fatal("expected error when no device found")
	}
}
