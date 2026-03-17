package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	cdpbrowser "github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"

	"github.com/ignatij/spotpilot/internal/app"
	browserintegration "github.com/ignatij/spotpilot/internal/integrations/browser"
)

const spotifyLoginURL = "https://accounts.spotify.com/login"

type cdpLoginPerformer struct {
	pollInterval time.Duration
}

func NewCDPLoginPerformer() app.LoginPerformer {
	return &cdpLoginPerformer{pollInterval: 2 * time.Second}
}

func (p *cdpLoginPerformer) PerformLogin(ctx context.Context) (*app.Session, error) {
	chromePath, err := browserintegration.FindChromeBinary()
	if err != nil {
		return nil, fmt.Errorf("login requires Chrome or Chromium: %w", err)
	}

	browserCtx, cleanup, err := launchLoginBrowser(ctx, chromePath)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	if err := chromedp.Run(browserCtx,
		network.Enable(),
		chromedp.Navigate(spotifyLoginURL),
	); err != nil {
		return nil, fmt.Errorf("opening Spotify login page: %w", err)
	}

	fmt.Println("Waiting for Spotify login... (sign in in the browser window, then wait)")

	for {
		cookies, err := p.spotifyCookies(browserCtx)
		if err == nil && len(cookies) > 0 {
			if !hasSpotifyCookie(cookies, "sp_t") {
				if refreshErr := p.populatePlaybackCookie(browserCtx); refreshErr == nil {
					if updated, cookieErr := p.spotifyCookies(browserCtx); cookieErr == nil {
						cookies = updated
					}
				}
			}
			return cookiesToSession(cookies), nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("login timed out or was cancelled: %w", ctx.Err())
		case <-time.After(p.pollInterval):
		}
	}
}

func launchLoginBrowser(parent context.Context, chromePath string) (context.Context, func(), error) {
	port, err := reserveTCPPort()
	if err != nil {
		return nil, nil, fmt.Errorf("allocating Chrome debug port: %w", err)
	}

	userDataDir, err := os.MkdirTemp("", "spotpilot-chrome-*")
	if err != nil {
		return nil, nil, fmt.Errorf("creating Chrome profile dir: %w", err)
	}
	if err := seedChromeProfile(userDataDir); err != nil {
		_ = os.RemoveAll(userDataDir)
		return nil, nil, err
	}

	procCtx, cancelProc := context.WithCancel(context.Background())
	cmd := exec.CommandContext(procCtx, chromePath,
		fmt.Sprintf("--remote-debugging-port=%d", port),
		fmt.Sprintf("--user-data-dir=%s", userDataDir),
		"--profile-directory=Default",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-default-apps",
		"about:blank",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		cancelProc()
		_ = os.RemoveAll(userDataDir)
		return nil, nil, fmt.Errorf("starting Chrome: %w", err)
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	if err := waitForChromeDevTools(parent, port, waitCh); err != nil {
		cancelProc()
		killProcessGroup(cmd.Process)
		<-waitCh
		_ = os.RemoveAll(userDataDir)
		return nil, nil, err
	}

	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(parent, fmt.Sprintf("http://127.0.0.1:%d", port))
	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)

	cleanup := func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer closeCancel()

		_ = closeChromeBrowser(closeCtx, browserCtx)
		cancelBrowser()
		cancelAlloc()
		cancelProc()
		killProcessGroup(cmd.Process)

		select {
		case <-waitCh:
		case <-time.After(2 * time.Second):
			killProcessGroup(cmd.Process)
			<-waitCh
		}

		_ = os.RemoveAll(userDataDir)
	}

	return browserCtx, cleanup, nil
}

func killProcessGroup(proc *os.Process) {
	if proc == nil {
		return
	}
	_ = syscall.Kill(-proc.Pid, syscall.SIGKILL)
}

func seedChromeProfile(tempUserDataDir string) error {
	sourceRoot, err := browserintegration.FindChromeUserDataDir()
	if err != nil {
		return nil
	}

	profileName, err := detectPreferredChromeProfile(sourceRoot)
	if err != nil {
		return fmt.Errorf("detecting Chrome profile: %w", err)
	}
	if profileName == "" {
		profileName = "Default"
	}

	if err := copyPath(filepath.Join(sourceRoot, "Local State"), filepath.Join(tempUserDataDir, "Local State")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("copying Chrome Local State: %w", err)
	}

	sourceProfileDir := filepath.Join(sourceRoot, profileName)
	targetProfileDir := filepath.Join(tempUserDataDir, "Default")
	for _, relative := range []string{"Cookies", "Cookies-journal", "Network Persistent State", "Preferences", "Local Storage", "Session Storage", "IndexedDB", "Storage", "Service Worker", "WebStorage"} {
		if err := copyPath(filepath.Join(sourceProfileDir, relative), filepath.Join(targetProfileDir, relative)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("copying Chrome profile data %s: %w", relative, err)
		}
	}

	return nil
}

func detectPreferredChromeProfile(userDataDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(userDataDir, "Local State"))
	if err != nil {
		if os.IsNotExist(err) {
			return "Default", nil
		}
		return "", err
	}

	var state struct {
		Profile struct {
			LastUsed string `json:"last_used"`
		} `json:"profile"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return "", err
	}
	if state.Profile.LastUsed == "" {
		return "Default", nil
	}
	return state.Profile.LastUsed, nil
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst, info.Mode())
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := copyPath(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func reserveTCPPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener address type %T", listener.Addr())
	}
	return addr.Port, nil
}

func waitForChromeDevTools(ctx context.Context, port int, waitCh <-chan error) error {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	url := fmt.Sprintf("http://127.0.0.1:%d/json/version", port)
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for Chrome startup cancelled: %w", ctx.Err())
		case err := <-waitCh:
			return fmt.Errorf("Chrome exited before devtools became available: %w", err)
		case <-timer.C:
			return fmt.Errorf("timed out waiting for Chrome remote debugging endpoint")
		case <-ticker.C:
		}
	}
}

func closeChromeBrowser(closeCtx context.Context, browserCtx context.Context) error {
	finished := make(chan error, 1)
	go func() {
		finished <- chromedp.Run(browserCtx, chromedp.ActionFunc(func(ctx context.Context) error {
			c := chromedp.FromContext(ctx)
			if c == nil || c.Browser == nil {
				return nil
			}
			return cdpbrowser.Close().Do(cdp.WithExecutor(ctx, c.Browser))
		}))
	}()

	select {
	case err := <-finished:
		return err
	case <-closeCtx.Done():
		return closeCtx.Err()
	}
}

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
	var hasSPDC bool
	for _, c := range all {
		if c.Domain != ".spotify.com" && c.Domain != "spotify.com" {
			continue
		}
		if c.Name == "sp_dc" || c.Name == "sp_key" || c.Name == "sp_t" {
			found = append(found, c)
			if c.Name == "sp_dc" {
				hasSPDC = true
			}
		}
	}
	if !hasSPDC {
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

func hasSpotifyCookie(cookies []*network.Cookie, name string) bool {
	for _, cookie := range cookies {
		if cookie.Name == name && cookie.Value != "" {
			return true
		}
	}
	return false
}

func (p *cdpLoginPerformer) populatePlaybackCookie(ctx context.Context) error {
	if err := chromedp.Run(ctx, chromedp.Navigate("https://open.spotify.com/")); err != nil {
		return err
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		cookies, err := p.spotifyCookies(ctx)
		if err == nil && hasSpotifyCookie(cookies, "sp_t") {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("sp_t cookie was not observed after opening the Spotify web player")
}
