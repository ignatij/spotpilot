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
	"strings"
	"sync"
	"syscall"
	"time"

	cdpbrowser "github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"github.com/ignatij/spotpilot/internal/app"
	browserintegration "github.com/ignatij/spotpilot/internal/integrations/browser"
)

const spotifyLoginURL = "https://accounts.spotify.com/login"

type cdpLoginPerformer struct {
	pollInterval time.Duration
}

type browserStorageSnapshot struct {
	LocalStorage   map[string]string `json:"localStorage"`
	SessionStorage map[string]string `json:"sessionStorage"`
	IndexedDB      []string          `json:"indexedDb"`
	IndexedDBData  map[string]string `json:"indexedDbData"`
	URL            string            `json:"url"`
	Origin         string            `json:"origin"`
	Error          string            `json:"error"`
}

type tokenObservation struct {
	Token string
	URL   string
}

type accessTokenResponse struct {
	AccessToken                      string `json:"accessToken"`
	AccessTokenExpirationTimestampMs int64  `json:"accessTokenExpirationTimestampMs"`
	IsAnonymous                      bool   `json:"isAnonymous"`
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

func (p *cdpLoginPerformer) fetchAccessToken(ctx context.Context) (string, time.Time, error) {
	observations := watchSpotifyAuthHeader(ctx)

	for _, targetURL := range []string{
		"https://open.spotify.com/",
		"https://open.spotify.com/search/spotify",
		"https://open.spotify.com/collection/tracks",
	} {
		navCtx, navCancel := context.WithTimeout(ctx, 15*time.Second)
		_ = chromedp.Run(navCtx, chromedp.Navigate(targetURL))
		navCancel()

		if err := waitForSpotifyOrigin(ctx, 10*time.Second); err != nil {
			return "", time.Time{}, err
		}
		if token, ok := waitForObservedAuthToken(ctx, observations, 8*time.Second); ok {
			return token, time.Now().Add(30 * time.Minute), nil
		}
	}

	if tok, expiry, err := extractTokenFromStorage(ctx); err == nil {
		return tok, expiry, nil
	}
	if snapshot, err := snapshotBrowserStorage(ctx); err == nil {
		return "", time.Time{}, fmt.Errorf("did not observe authenticated Spotify API traffic; storage keys: %s", describeStorageSnapshot(snapshot))
	}
	return "", time.Time{}, fmt.Errorf("did not observe authenticated Spotify API traffic")
}

func waitForSpotifyOrigin(ctx context.Context, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		var origin string
		err := chromedp.Run(ctx, chromedp.Evaluate(`window.location.origin`, &origin))
		if err == nil && origin == "https://open.spotify.com" {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for Spotify web player: %w", ctx.Err())
		case <-timer.C:
			if err != nil {
				return fmt.Errorf("timed out waiting for Spotify web player origin: %w", err)
			}
			return fmt.Errorf("timed out waiting for Spotify web player origin; current origin was %q", origin)
		case <-ticker.C:
		}
	}
}

func waitForObservedAuthToken(ctx context.Context, observations <-chan tokenObservation, timeout time.Duration) (string, bool) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	seen := make(map[string]struct{})

	for {
		select {
		case obs := <-observations:
			if obs.Token == "" {
				continue
			}
			if _, ok := seen[obs.Token]; ok {
				continue
			}
			seen[obs.Token] = struct{}{}
			if isSpotifyAuthorizedAPIURL(obs.URL) {
				return obs.Token, true
			}
		case <-deadline.C:
			return "", false
		case <-ctx.Done():
			return "", false
		}
	}
}

func watchSpotifyAuthHeader(ctx context.Context) <-chan tokenObservation {
	tokens := make(chan tokenObservation, 8)
	var requestURLs sync.Map

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			requestURLs.Store(e.RequestID, e.Request.URL)
		case *network.EventRequestWillBeSentExtraInfo:
			tok, ok := bearerTokenFromHeaders(e.Headers)
			if !ok {
				return
			}
			urlVal, _ := requestURLs.Load(e.RequestID)
			url, _ := urlVal.(string)
			if url != "" && !isSpotifyAuthorizedAPIURL(url) {
				return
			}
			select {
			case tokens <- tokenObservation{Token: tok, URL: url}:
			default:
			}
		}
	})

	return tokens
}

func bearerTokenFromHeaders(headers network.Headers) (string, bool) {
	for key, raw := range headers {
		if !strings.EqualFold(key, "authorization") {
			continue
		}
		value, ok := raw.(string)
		if !ok {
			continue
		}
		const prefix = "Bearer "
		if strings.HasPrefix(value, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(value, prefix)), true
		}
	}
	return "", false
}

func isSpotifyAuthorizedAPIURL(url string) bool {
	return strings.Contains(url, "api.spotify.com") || strings.Contains(url, "api-partner.spotify.com") || strings.Contains(url, "spclient.wg.spotify.com")
}

func extractTokenFromStorage(ctx context.Context) (string, time.Time, error) {
	snapshot, err := snapshotBrowserStorage(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	for _, storage := range []map[string]string{snapshot.LocalStorage, snapshot.SessionStorage, snapshot.IndexedDBData} {
		for _, raw := range storage {
			if tok, expiry, ok := parseTokenCandidate(raw); ok {
				return tok, expiry, nil
			}
		}
	}
	return "", time.Time{}, fmt.Errorf("no usable access token found in browser storage")
}

func snapshotBrowserStorage(ctx context.Context) (*browserStorageSnapshot, error) {
	const expr = `(async () => {
		const dumpIndexedDb = async () => {
			const out = {};
			if (!indexedDB || !indexedDB.databases) {
				return out;
			}
			const dbs = await indexedDB.databases();
			for (const dbInfo of dbs) {
				if (!dbInfo.name) {
					continue;
				}
				const dbName = dbInfo.name;
				const db = await new Promise((resolve, reject) => {
					const req = indexedDB.open(dbName);
					req.onsuccess = () => resolve(req.result);
					req.onerror = () => reject(req.error);
				});
				try {
					for (const storeName of Array.from(db.objectStoreNames)) {
						const storeKey = dbName + "/" + storeName;
						const records = await new Promise((resolve, reject) => {
							const tx = db.transaction(storeName, "readonly");
							const store = tx.objectStore(storeName);
							const req = store.getAll();
							req.onsuccess = () => resolve(req.result || []);
							req.onerror = () => reject(req.error);
						});
						out[storeKey] = JSON.stringify(records).slice(0, 20000);
					}
				} finally {
					db.close();
				}
			}
			return out;
		};

		try {
			const local = {};
			for (let i = 0; i < window.localStorage.length; i += 1) {
				const key = window.localStorage.key(i);
				local[key] = window.localStorage.getItem(key) ?? "";
			}
			const session = {};
			for (let i = 0; i < window.sessionStorage.length; i += 1) {
				const key = window.sessionStorage.key(i);
				session[key] = window.sessionStorage.getItem(key) ?? "";
			}
			let indexedDb = [];
			if (indexedDB && indexedDB.databases) {
				const dbs = await indexedDB.databases();
				indexedDb = dbs.map((db) => db.name || "");
			}
			const indexedDbData = await dumpIndexedDb();
			return { localStorage: local, sessionStorage: session, indexedDb, indexedDbData, url: window.location.href, origin: window.location.origin, error: "" };
		} catch (err) {
			return { localStorage: {}, sessionStorage: {}, indexedDb: [], indexedDbData: {}, url: window.location.href, origin: window.location.origin, error: String(err) };
		}
	})()`

	var snapshot browserStorageSnapshot
	if err := chromedp.Run(ctx, chromedp.Evaluate(expr, &snapshot, chromedp.EvalAsValue, evalAwaitPromise)); err != nil {
		return nil, err
	}
	if snapshot.Error != "" {
		return nil, fmt.Errorf(snapshot.Error)
	}
	return &snapshot, nil
}

func parseTokenCandidate(raw string) (string, time.Time, bool) {
	if raw == "" {
		return "", time.Time{}, false
	}

	var direct accessTokenResponse
	if json.Unmarshal([]byte(raw), &direct) == nil && direct.AccessToken != "" && !direct.IsAnonymous {
		return direct.AccessToken, time.UnixMilli(direct.AccessTokenExpirationTimestampMs), true
	}

	var generic map[string]any
	if json.Unmarshal([]byte(raw), &generic) != nil {
		return "", time.Time{}, false
	}

	tok, ok := findStringRecursive(generic, "accessToken", "access_token")
	if !ok || tok == "" {
		return "", time.Time{}, false
	}
	if anon, ok := findBoolRecursive(generic, "isAnonymous", "is_anonymous"); ok && anon {
		return "", time.Time{}, false
	}
	if expiryMs, ok := findInt64Recursive(generic, "accessTokenExpirationTimestampMs", "access_token_expiration_timestamp_ms", "expiration_timestamp_ms", "expiresAt", "expires_at"); ok {
		return tok, time.UnixMilli(expiryMs), true
	}
	if expiresIn, ok := findInt64Recursive(generic, "expiresIn", "expires_in"); ok {
		return tok, time.Now().Add(time.Duration(expiresIn) * time.Second), true
	}
	return tok, time.Now().Add(30 * time.Minute), true
}

func findStringRecursive(value any, keys ...string) (string, bool) {
	switch v := value.(type) {
	case map[string]any:
		for _, key := range keys {
			if raw, ok := v[key]; ok {
				if s, ok := raw.(string); ok {
					return s, true
				}
			}
		}
		for _, child := range v {
			if s, ok := findStringRecursive(child, keys...); ok {
				return s, true
			}
		}
	case []any:
		for _, child := range v {
			if s, ok := findStringRecursive(child, keys...); ok {
				return s, true
			}
		}
	}
	return "", false
}

func findBoolRecursive(value any, keys ...string) (bool, bool) {
	switch v := value.(type) {
	case map[string]any:
		for _, key := range keys {
			if raw, ok := v[key]; ok {
				if b, ok := raw.(bool); ok {
					return b, true
				}
			}
		}
		for _, child := range v {
			if b, ok := findBoolRecursive(child, keys...); ok {
				return b, true
			}
		}
	case []any:
		for _, child := range v {
			if b, ok := findBoolRecursive(child, keys...); ok {
				return b, true
			}
		}
	}
	return false, false
}

func findInt64Recursive(value any, keys ...string) (int64, bool) {
	switch v := value.(type) {
	case map[string]any:
		for _, key := range keys {
			if raw, ok := v[key]; ok {
				switch n := raw.(type) {
				case float64:
					return int64(n), true
				case int64:
					return n, true
				case int:
					return int64(n), true
				}
			}
		}
		for _, child := range v {
			if n, ok := findInt64Recursive(child, keys...); ok {
				return n, true
			}
		}
	case []any:
		for _, child := range v {
			if n, ok := findInt64Recursive(child, keys...); ok {
				return n, true
			}
		}
	}
	return 0, false
}

func describeStorageSnapshot(snapshot *browserStorageSnapshot) string {
	if snapshot == nil {
		return "none"
	}
	return fmt.Sprintf("local=%v session=%v indexeddb=%v indexeddbStores=%v", mapKeys(snapshot.LocalStorage), mapKeys(snapshot.SessionStorage), snapshot.IndexedDB, mapKeys(snapshot.IndexedDBData))
}

func mapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func evalAwaitPromise(p *runtime.EvaluateParams) *runtime.EvaluateParams {
	return p.WithAwaitPromise(true)
}
