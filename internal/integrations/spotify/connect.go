package spotify

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"

	"github.com/ignatij/spotpilot/internal/domain"
)

const (
	connectStateBase   = "https://gue1-spclient.spotify.com/connect-state/v1"
	trackPlaybackBase  = "https://gue1-spclient.spotify.com/track-playback/v1"
	dealerURL          = "wss://dealer.spotify.com/"
	connectionTTL      = 10 * time.Minute
	maxWebRetryAfter   = 2 * time.Second
	defaultAppPlatform = "WebPlayer"
	connectDeviceName  = "spotpilot"
	connectDeviceModel = "web_player"
	connectVersionEnv  = "SPOTPILOT_CONNECT_VERSION"
)

type HybridClient struct {
	web     *Client
	connect *ConnectClient
}

type ConnectClient struct {
	web     *Client
	http    *http.Client
	source  CookieSource
	session *connectSession
	debug   debugLogger
}

type connectSession struct {
	source CookieSource
	http   *http.Client

	mu sync.Mutex

	token           Token
	clientToken     string
	clientTokenTill time.Time
	clientID        string
	clientVersion   string
	connectVersion  string
	deviceID        string

	connectDeviceID string
	connectionID    string
	registeredAt    time.Time
}

type connectAuth struct {
	AccessToken    string
	ClientToken    string
	ClientVersion  string
	ConnectVersion string
	DeviceID       string
}

type connectState struct {
	playerState    map[string]any
	devices        map[string]any
	activeDeviceID string
	originDeviceID string
}

type requestHeaders struct {
	AccessToken   string
	ClientToken   string
	ClientVersion string
	Accept        string
	ContentType   string
	Language      string
	AppPlatform   string
	ConnectionID  string
}

func NewConnectClient(source CookieSource, web *Client) (*ConnectClient, error) {
	if source == nil {
		return nil, errors.New("cookie source required")
	}
	httpClient := &http.Client{Timeout: 10 * time.Second}
	session := &connectSession{source: source, http: httpClient}
	return &ConnectClient{web: web, http: httpClient, source: source, session: session, debug: web.debug}, nil
}

func NewHybridClient(web *Client, connect *ConnectClient) *HybridClient {
	return &HybridClient{web: web, connect: connect}
}

func (c *HybridClient) Search(ctx context.Context, query string) (*domain.MatchResult, error) {
	c.web.debug.printf("hybrid search start query=%q", query)
	webMatch, webErr := c.web.Search(ctx, query)
	if webErr == nil && webMatch != nil {
		c.web.debug.printf("hybrid search using web result type=%s uri=%s", webMatch.Type, webMatch.PlaybackURI())
		return webMatch, nil
	}
	if webErr != nil {
		c.web.debug.printf("hybrid search web error for query=%q: %v", query, webErr)
	} else {
		c.web.debug.printf("hybrid search web returned no match for query=%q", query)
	}

	connectMatch, connectErr := c.connect.Search(ctx, query)
	if connectErr == nil {
		if connectMatch != nil {
			c.web.debug.printf("hybrid search using connect result type=%s uri=%s", connectMatch.Type, connectMatch.PlaybackURI())
		} else {
			c.web.debug.printf("hybrid search connect returned nil match for query=%q", query)
		}
		return connectMatch, nil
	}
	c.web.debug.printf("hybrid search connect error for query=%q: %v", query, connectErr)

	return nil, fmt.Errorf("spotify hybrid search: web=%v connect=%w", webErr, connectErr)
}

func (c *HybridClient) Play(ctx context.Context, deviceID string, uri string) error {
	webErr := c.playViaWebWithRetry(ctx, deviceID, uri)
	if webErr == nil {
		return nil
	}

	err := c.connect.Play(ctx, deviceID, uri)
	if err == nil {
		return nil
	}
	if isGoneError(err) {
		c.connect.invalidateRegistration()
		retryErr := c.connect.Play(ctx, deviceID, uri)
		if retryErr == nil {
			return nil
		}
		err = fmt.Errorf("%v; reconnect_retry=%v", err, retryErr)
	}

	if isRateLimitedError(webErr) || isGoneError(err) || isRateLimitedError(err) {
		return fmt.Errorf("web API failed: %v; connect failed: %v", webErr, err)
	}

	return err
}

func (c *HybridClient) playViaWebWithRetry(ctx context.Context, deviceID, uri string) error {
	const maxAttempts = 2
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := c.web.Play(ctx, deviceID, uri)
		if err == nil {
			return nil
		}
		lastErr = err

		retryAfter, ok := retryAfterFromError(err)
		if !ok || retryAfter <= 0 || retryAfter > maxWebRetryAfter || !canWaitForRetryAfter(ctx, retryAfter) {
			break
		}
		if waitErr := waitRetryAfter(ctx, retryAfter); waitErr != nil {
			lastErr = fmt.Errorf("%v; retry_wait=%v", err, waitErr)
			break
		}
	}
	return lastErr
}

func (c *ConnectClient) invalidateRegistration() {
	c.session.mu.Lock()
	c.session.connectionID = ""
	c.session.registeredAt = time.Time{}
	c.session.connectDeviceID = randomHex()
	c.session.mu.Unlock()
}

func (c *HybridClient) Pause(ctx context.Context, deviceID string) error {
	return c.connect.Pause(ctx, deviceID)
}

func (c *HybridClient) Resume(ctx context.Context, deviceID string) error {
	return c.connect.Resume(ctx, deviceID)
}

func (c *HybridClient) Next(ctx context.Context, deviceID string) error {
	return c.connect.Next(ctx, deviceID)
}

func (c *HybridClient) Previous(ctx context.Context, deviceID string) error {
	return c.connect.Previous(ctx, deviceID)
}

func (c *HybridClient) CurrentPlayback(ctx context.Context) (*domain.CurrentPlayback, error) {
	playback, err := c.web.CurrentPlayback(ctx)
	if err == nil {
		return playback, nil
	}
	return c.connect.CurrentPlayback(ctx)
}

func (c *HybridClient) ListDevices(ctx context.Context) ([]domain.Device, error) {
	devices, err := c.connect.ListDevices(ctx)
	if err == nil {
		return devices, nil
	}
	return c.web.ListDevices(ctx)
}

func (c *ConnectClient) Search(ctx context.Context, query string) (*domain.MatchResult, error) {
	c.debug.printf("connect search start query=%q", query)
	match, err := c.searchViaSearchView(ctx, query)
	if err != nil {
		c.debug.printf("connect search searchview failed for query=%q: %v", query, err)
		return nil, fmt.Errorf("spotify connect search: %w", err)
	}
	if match != nil {
		c.debug.printf("connect search searchview matched type=%s uri=%s", match.Type, match.PlaybackURI())
	} else {
		c.debug.printf("connect search searchview returned nil match for query=%q", query)
	}
	return match, nil
}

func (c *ConnectClient) searchViaSearchView(ctx context.Context, query string) (*domain.MatchResult, error) {
	auth, err := c.session.auth(ctx)
	if err != nil {
		return nil, err
	}

	endpoint := "https://spclient.wg.spotify.com/searchview/km/v4/search/" + url.PathEscape(query)
	params := url.Values{}
	params.Set("entityVersion", "2")
	params.Set("limit", "1")
	params.Set("imageSize", "small")
	params.Set("country", "from_token")
	params.Set("locale", "en")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("spotify searchview: %w", err)
	}
	applyRequestHeaders(req, requestHeaders{
		AccessToken:   auth.AccessToken,
		ClientToken:   auth.ClientToken,
		ClientVersion: connectVersionFor(auth),
		Accept:        "application/json",
		Language:      "en-US,en;q=0.9",
		AppPlatform:   defaultAppPlatform,
	})

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("spotify searchview: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify searchview: %w", apiError(resp))
	}

	var payload any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("spotify searchview decode: %w", err)
	}
	c.debug.dumpJSON(fmt.Sprintf("searchview payload for query %q", query), payload)
	return matchResultFromSearchviewPayload(payload), nil
}

func matchResultFromSearchviewPayload(payload any) *domain.MatchResult {
	if mapped, ok := payload.(map[string]any); ok {
		if results, ok := mapped["results"].(map[string]any); ok {
			if topHit, ok := results["topHit"].(map[string]any); ok {
				if hits, ok := topHit["hits"].([]any); ok && len(hits) > 0 {
					if hit, ok := hits[0].(map[string]any); ok {
						if uri, _ := hit["uri"].(string); uri != "" {
							return matchResultFromURI(uri, hit)
						}
					}
				}
			}
		}
	}
	uris := make([]string, 0, 8)
	collectSpotifyURIs(payload, &uris)
	for _, uri := range uris {
		if result := matchResultFromURI(uri, nil); result != nil {
			return result
		}
	}
	return nil
}

func matchResultFromURI(uri string, hit map[string]any) *domain.MatchResult {
	switch {
	case strings.HasPrefix(uri, "spotify:track:"):
		track := &domain.Track{URI: uri}
		if hit != nil {
			track.Title, _ = hit["name"].(string)
			track.Artist = searchviewHitArtist(hit)
			if album, ok := hit["album"].(map[string]any); ok {
				track.Album, _ = album["name"].(string)
			}
		}
		return &domain.MatchResult{Type: domain.MatchTypeTrack, Track: track}
	case strings.HasPrefix(uri, "spotify:album:"):
		album := &domain.Album{URI: uri}
		if hit != nil {
			album.Name, _ = hit["name"].(string)
			album.Artist = searchviewHitArtist(hit)
		}
		return &domain.MatchResult{Type: domain.MatchTypeAlbum, Album: album}
	case strings.HasPrefix(uri, "spotify:artist:"):
		artist := &domain.Artist{URI: uri}
		if hit != nil {
			artist.Name, _ = hit["name"].(string)
		}
		return &domain.MatchResult{Type: domain.MatchTypeArtist, Artist: artist}
	}
	return nil
}

func searchviewHitArtist(hit map[string]any) string {
	artists, ok := hit["artists"].([]any)
	if !ok || len(artists) == 0 {
		return ""
	}
	if artist, ok := artists[0].(map[string]any); ok {
		name, _ := artist["name"].(string)
		return name
	}
	return ""
}

func collectSpotifyURIs(value any, uris *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		for _, nested := range typed {
			collectSpotifyURIs(nested, uris)
		}
	case []any:
		for _, nested := range typed {
			collectSpotifyURIs(nested, uris)
		}
	case string:
		if strings.HasPrefix(typed, "spotify:track:") || strings.HasPrefix(typed, "spotify:album:") || strings.HasPrefix(typed, "spotify:artist:") {
			*uris = append(*uris, typed)
		}
	}
}

func (c *ConnectClient) Play(ctx context.Context, deviceID string, uri string) error {
	state, err := c.connectState(ctx)
	if err != nil {
		return err
	}
	targetDeviceID := strings.TrimSpace(deviceID)
	if targetDeviceID == "" {
		targetDeviceID = state.activeDeviceID
	}
	if targetDeviceID == "" {
		return errors.New("missing device id")
	}
	endpoint := "play"
	payload := playCommandPayload(uri)
	if uri == "" {
		endpoint = "resume"
		payload = nil
	}

	err = c.sendPlayerCommandTo(ctx, state, targetDeviceID, endpoint, payload)
	if err == nil {
		return nil
	}
	if !isGoneError(err) && !isNotFoundError(err) {
		return err
	}

	transferErr := c.Transfer(ctx, targetDeviceID)
	refreshed, stateErr := c.connectState(ctx)
	if transferErr == nil && stateErr == nil {
		retryErr := c.sendPlayerCommandTo(ctx, refreshed, targetDeviceID, endpoint, payload)
		if retryErr == nil {
			return nil
		}
		return fmt.Errorf("%v; transfer_retry=%v", err, retryErr)
	}
	if transferErr != nil && stateErr != nil {
		return fmt.Errorf("%v; transfer=%v; refresh_state=%v", err, transferErr, stateErr)
	}
	if transferErr != nil {
		return fmt.Errorf("%v; transfer=%v", err, transferErr)
	}
	return fmt.Errorf("%v; refresh_state=%v", err, stateErr)
}

func (c *ConnectClient) Pause(ctx context.Context, deviceID string) error {
	state, err := c.connectState(ctx)
	if err != nil {
		return err
	}
	return c.sendPlayerCommandTo(ctx, state, deviceID, "pause", nil)
}

func (c *ConnectClient) Resume(ctx context.Context, deviceID string) error {
	state, err := c.connectState(ctx)
	if err != nil {
		return err
	}
	return c.sendPlayerCommandTo(ctx, state, deviceID, "resume", nil)
}

func (c *ConnectClient) Next(ctx context.Context, deviceID string) error {
	state, err := c.connectState(ctx)
	if err != nil {
		return err
	}
	return c.sendPlayerCommandTo(ctx, state, deviceID, "skip_next", nil)
}

func (c *ConnectClient) Previous(ctx context.Context, deviceID string) error {
	state, err := c.connectState(ctx)
	if err != nil {
		return err
	}
	return c.sendPlayerCommandTo(ctx, state, deviceID, "skip_prev", nil)
}

func (c *ConnectClient) CurrentPlayback(ctx context.Context) (*domain.CurrentPlayback, error) {
	state, err := c.connectState(ctx)
	if err != nil {
		return nil, err
	}
	return mapPlaybackState(state), nil
}

func (c *ConnectClient) ListDevices(ctx context.Context) ([]domain.Device, error) {
	state, err := c.connectState(ctx)
	if err != nil {
		return nil, err
	}
	return mapDevices(state.devices), nil
}

func (c *ConnectClient) Transfer(ctx context.Context, deviceID string) error {
	if strings.TrimSpace(deviceID) == "" {
		return nil
	}
	state, err := c.connectState(ctx)
	if err != nil {
		return err
	}
	fromID := state.originDeviceID
	if fromID == "" {
		fromID = state.activeDeviceID
	}
	if fromID == "" {
		if c.web == nil {
			return errors.New("missing active Spotify device")
		}
		return c.web.transferPlayback(ctx, deviceID)
	}
	return c.sendConnectCommand(ctx,
		fmt.Sprintf("%s/connect/transfer/from/%s/to/%s", connectStateBase, fromID, deviceID),
		map[string]any{
			"transfer_options": map[string]any{"restore_paused": "resume"},
			"command_id":       randomHex(),
		},
	)
}

func (c *ConnectClient) connectState(ctx context.Context) (connectState, error) {
	auth, err := c.session.auth(ctx)
	if err != nil {
		return connectState{}, err
	}
	if err := c.ensureConnectDevice(ctx, auth); err != nil {
		return connectState{}, err
	}

	c.session.mu.Lock()
	deviceID := c.session.connectDeviceID
	connectionID := c.session.connectionID
	c.session.mu.Unlock()

	payload := map[string]any{
		"member_type": "CONNECT_STATE",
		"device": map[string]any{
			"device_info": map[string]any{
				"capabilities": map[string]any{
					"can_be_player":           false,
					"hidden":                  true,
					"needs_full_player_state": true,
				},
			},
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/devices/hobs_%s", connectStateBase, deviceID), encodeJSON(payload))
	if err != nil {
		return connectState{}, err
	}
	applyRequestHeaders(req, requestHeaders{
		AccessToken:   auth.AccessToken,
		ClientToken:   auth.ClientToken,
		ClientVersion: connectVersionFor(auth),
		ContentType:   "application/json",
		AppPlatform:   defaultAppPlatform,
		ConnectionID:  connectionID,
	})

	resp, err := c.http.Do(req)
	if err != nil {
		return connectState{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return connectState{}, fmt.Errorf("spotify connect state: %w", apiError(resp))
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return connectState{}, err
	}

	state := connectState{}
	if devices, ok := raw["devices"].(map[string]any); ok {
		state.devices = devices
	}
	if playerState, ok := raw["player_state"].(map[string]any); ok {
		state.playerState = playerState
	}
	if active, ok := raw["active_device_id"].(string); ok {
		state.activeDeviceID = active
	}
	if state.activeDeviceID == "" {
		state.activeDeviceID = detectActiveDeviceID(state.devices)
	}
	state.originDeviceID = mapPlayOriginID(state.playerState)
	return state, nil
}

func (c *ConnectClient) ensureConnectDevice(ctx context.Context, auth connectAuth) error {
	c.session.mu.Lock()
	if c.session.connectDeviceID == "" {
		c.session.connectDeviceID = randomHex()
	}
	needsRegistration := c.session.connectionID == "" || time.Since(c.session.registeredAt) > connectionTTL
	c.session.mu.Unlock()
	if !needsRegistration {
		return nil
	}

	connectionID, err := getConnectionID(ctx, auth.AccessToken)
	if err != nil {
		return err
	}
	if err := c.registerDevice(ctx, auth, connectionID); err != nil {
		return err
	}

	c.session.mu.Lock()
	c.session.connectionID = connectionID
	c.session.registeredAt = time.Now()
	c.session.mu.Unlock()
	return nil
}

func (c *ConnectClient) registerDevice(ctx context.Context, auth connectAuth, connectionID string) error {
	c.session.mu.Lock()
	deviceID := c.session.connectDeviceID
	c.session.mu.Unlock()

	payload := map[string]any{
		"device": map[string]any{
			"device_id":           deviceID,
			"device_type":         "computer",
			"brand":               "spotify",
			"model":               connectDeviceModel,
			"name":                connectDeviceName,
			"is_group":            false,
			"metadata":            map[string]any{},
			"platform_identifier": fmt.Sprintf("web_player %s;spotpilot", runtime.GOOS),
			"capabilities": map[string]any{
				"change_volume":            true,
				"supports_file_media_type": true,
				"enable_play_token":        true,
				"play_token_lost_behavior": "pause",
				"disable_connect":          false,
				"audio_podcasts":           true,
				"video_playback":           true,
				"manifest_formats": []string{
					"file_ids_mp3",
					"file_urls_mp3",
					"file_ids_mp4",
					"manifest_ids_video",
				},
			},
		},
		"outro_endcontent_snooping": false,
		"connection_id":             connectionID,
		"client_version":            connectVersionFor(auth),
		"volume":                    65535,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, trackPlaybackBase+"/devices", encodeJSON(payload))
	if err != nil {
		return err
	}
	applyRequestHeaders(req, requestHeaders{
		AccessToken:   auth.AccessToken,
		ClientToken:   auth.ClientToken,
		ClientVersion: connectVersionFor(auth),
		ContentType:   "application/json",
		AppPlatform:   defaultAppPlatform,
	})

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("spotify connect register device: %w", apiError(resp))
	}
	return nil
}

func (c *ConnectClient) sendPlayerCommandTo(ctx context.Context, state connectState, deviceID string, endpoint string, payload map[string]any) error {
	if payload == nil {
		payload = map[string]any{
			"command": map[string]any{
				"endpoint": endpoint,
				"logging_params": map[string]any{
					"command_id": randomHex(),
				},
			},
		}
	}

	fromID := state.originDeviceID
	if fromID == "" {
		c.session.mu.Lock()
		fromID = c.session.connectDeviceID
		c.session.mu.Unlock()
	}
	if fromID == "" {
		fromID = state.activeDeviceID
	}
	toID := strings.TrimSpace(deviceID)
	if toID == "" {
		toID = state.activeDeviceID
	}
	if fromID == "" || toID == "" {
		return errors.New("missing device id")
	}

	return c.sendConnectCommand(ctx,
		fmt.Sprintf("%s/player/command/from/%s/to/%s", connectStateBase, fromID, toID),
		payload,
	)
}

func (c *ConnectClient) sendConnectCommand(ctx context.Context, url string, payload map[string]any) error {
	auth, err := c.session.auth(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, encodeJSON(payload))
	if err != nil {
		return err
	}
	applyRequestHeaders(req, requestHeaders{
		AccessToken:   auth.AccessToken,
		ClientToken:   auth.ClientToken,
		ClientVersion: connectVersionFor(auth),
		ContentType:   "application/json",
		AppPlatform:   defaultAppPlatform,
	})

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("spotify connect command: %w", apiError(resp))
	}
	return nil
}

func (s *connectSession) auth(ctx context.Context) (connectAuth, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureTokenLocked(ctx); err != nil {
		return connectAuth{}, err
	}
	if err := s.ensureAppConfigLocked(ctx); err != nil {
		return connectAuth{}, err
	}
	if err := s.ensureClientTokenLocked(ctx); err != nil {
		return connectAuth{}, err
	}

	return connectAuth{
		AccessToken:    s.token.AccessToken,
		ClientToken:    s.clientToken,
		ClientVersion:  s.clientVersion,
		ConnectVersion: s.connectVersion,
		DeviceID:       s.deviceID,
	}, nil
}

func (s *connectSession) ensureTokenLocked(ctx context.Context) error {
	if s.token.AccessToken != "" && time.Until(s.token.ExpiresAt) > time.Minute {
		return nil
	}
	token, err := (CookieTokenProvider{Source: s.source, Client: s.http}).Token(ctx)
	if err != nil {
		return err
	}
	s.token = token
	if token.ClientID != "" {
		s.clientID = token.ClientID
	}
	return nil
}

func (s *connectSession) ensureAppConfigLocked(ctx context.Context) error {
	if s.clientVersion != "" && s.deviceID != "" {
		return nil
	}
	cookies, err := s.source.Cookies(ctx)
	if err != nil {
		return err
	}

	deviceID := ""
	for _, cookie := range cookies {
		if cookie.Name == "sp_t" {
			deviceID = cookie.Value
			break
		}
	}
	if deviceID == "" {
		return errors.New("missing sp_t cookie; run 'spotpilot login' again")
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	baseURL, _ := url.Parse(defaultTokenBaseURL)
	jar.SetCookies(baseURL, cookies)

	client := *s.http
	client.Jar = jar
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, defaultTokenBaseURL, nil)
	if err != nil {
		return err
	}
	applyRequestHeaders(req, requestHeaders{})

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("spotify app config: %w", apiError(resp))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	match := regexp.MustCompile(`<script id="appServerConfig" type="text/plain">([^<]+)</script>`).FindStringSubmatch(string(body))
	if len(match) < 2 {
		return errors.New("missing appServerConfig")
	}
	raw, err := base64.StdEncoding.DecodeString(match[1])
	if err != nil {
		return err
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	clientVersion, _ := payload["clientVersion"].(string)
	if clientVersion == "" {
		return errors.New("missing client version")
	}
	if index := strings.Index(clientVersion, ".g"); index > 0 {
		clientVersion = clientVersion[:index]
	}

	s.clientVersion = clientVersion
	s.connectVersion = resolveConnectVersion()
	s.deviceID = deviceID
	return nil
}

func (s *connectSession) ensureClientTokenLocked(ctx context.Context) error {
	if s.clientToken != "" && time.Until(s.clientTokenTill) > time.Minute {
		return nil
	}
	if s.clientID == "" {
		return errors.New("missing client id")
	}

	osName := runtimeOS()
	payload := map[string]any{
		"client_data": map[string]any{
			"client_version": s.clientVersion,
			"client_id":      s.clientID,
			"js_sdk_data": map[string]any{
				"device_brand": "unknown",
				"device_model": "unknown",
				"os":           osName,
				"os_version":   "unknown",
				"device_id":    s.deviceID,
				"device_type":  "computer",
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://clienttoken.spotify.com/v1/clienttoken", bytes.NewReader(body))
	if err != nil {
		return err
	}
	applyRequestHeaders(req, requestHeaders{ContentType: "application/json", Accept: "application/json"})

	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("spotify client token: %w", apiError(resp))
	}

	var tokenPayload struct {
		GrantedToken struct {
			Token   string `json:"token"`
			Expires int    `json:"expires_in"`
		} `json:"granted_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenPayload); err != nil {
		return err
	}
	if tokenPayload.GrantedToken.Token == "" {
		return errors.New("missing client token")
	}

	s.clientToken = tokenPayload.GrantedToken.Token
	if tokenPayload.GrantedToken.Expires > 0 {
		s.clientTokenTill = time.Now().Add(time.Duration(tokenPayload.GrantedToken.Expires) * time.Second)
	} else {
		s.clientTokenTill = time.Now().Add(30 * time.Minute)
	}
	return nil
}

func getConnectionID(ctx context.Context, accessToken string) (string, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	endpoint := dealerURL + "?access_token=" + url.QueryEscape(accessToken)
	conn, _, _, err := ws.Dial(dialCtx, endpoint)
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()

	msg, err := wsutil.ReadServerText(conn)
	if err != nil {
		return "", err
	}

	var payload map[string]any
	if err := json.Unmarshal(msg, &payload); err != nil {
		return "", err
	}
	headers, ok := payload["headers"].(map[string]any)
	if !ok {
		return "", errors.New("missing headers")
	}
	for key, value := range headers {
		if !strings.EqualFold(key, "Spotify-Connection-Id") {
			continue
		}
		if connectionID, ok := value.(string); ok && connectionID != "" {
			return connectionID, nil
		}
	}
	return "", errors.New("missing connection id")
}

func applyRequestHeaders(req *http.Request, headers requestHeaders) {
	if req == nil {
		return
	}
	req.Header.Set("User-Agent", defaultUserAgent())
	if headers.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+headers.AccessToken)
	}
	if headers.ClientToken != "" {
		req.Header.Set("Client-Token", headers.ClientToken)
	}
	if headers.ClientVersion != "" {
		req.Header.Set("Spotify-App-Version", headers.ClientVersion)
	}
	if headers.Accept != "" {
		req.Header.Set("Accept", headers.Accept)
	}
	if headers.ContentType != "" {
		req.Header.Set("Content-Type", headers.ContentType)
	}
	if headers.Language != "" {
		req.Header.Set("Accept-Language", headers.Language)
	}
	if headers.AppPlatform != "" {
		req.Header.Set("app-platform", headers.AppPlatform)
	}
	if headers.ConnectionID != "" {
		req.Header.Set("x-spotify-connection-id", headers.ConnectionID)
	}
}

func defaultUserAgent() string {
	return "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
}

func isRateLimitedError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "status 429") || strings.Contains(text, "too many requests")
}

func isGoneError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "status 410") || strings.Contains(text, "gone")
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "status 404") || strings.Contains(text, "not found")
}

func retryAfterFromError(err error) (time.Duration, bool) {
	if err == nil {
		return 0, false
	}
	match := regexp.MustCompile(`retry_after=([0-9]+(?:\.[0-9]+)?[smh])`).FindStringSubmatch(strings.ToLower(err.Error()))
	if len(match) != 2 {
		return 0, false
	}
	d, parseErr := time.ParseDuration(match[1])
	if parseErr != nil || d <= 0 {
		return 0, false
	}
	return d, true
}

func waitRetryAfter(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func canWaitForRetryAfter(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return false
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return d <= maxWebRetryAfter
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return false
	}
	// Keep a small buffer so command processing still has time after the wait.
	return d+2*time.Second <= remaining
}

func resolveConnectVersion() string {
	if override := strings.TrimSpace(os.Getenv(connectVersionEnv)); override != "" {
		return override
	}
	return "harmony:4.43.2-a61ecaf5"
}

func runtimeOS() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	case "windows":
		return "windows"
	default:
		return "linux"
	}
}

func encodeJSON(payload any) *strings.Reader {
	if payload == nil {
		return strings.NewReader("{}")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return strings.NewReader("{}")
	}
	return strings.NewReader(string(data))
}

func randomHex() string {
	const size = 32
	raw := make([]byte, size/2)
	if _, err := rand.Read(raw); err != nil {
		return strings.Repeat("0", size)
	}
	return hex.EncodeToString(raw)
}

func connectVersionFor(auth connectAuth) string {
	if auth.ConnectVersion != "" {
		return auth.ConnectVersion
	}
	return auth.ClientVersion
}

func playCommandPayload(uri string) map[string]any {
	command := map[string]any{
		"endpoint": "play",
		"logging_params": map[string]any{
			"command_id": randomHex(),
		},
		"context": map[string]any{
			"uri": uri,
			"url": "context://" + uri,
		},
	}
	if !isContextURI(uri) {
		command["options"] = map[string]any{
			"skip_to": map[string]any{"track_uri": uri},
		}
	}
	return map[string]any{"command": command}
}

func isContextURI(uri string) bool {
	return !strings.HasPrefix(uri, "spotify:track:")
}

func detectActiveDeviceID(devices map[string]any) string {
	for id, raw := range devices {
		device, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if active, _ := device["is_active"].(bool); active {
			return id
		}
	}
	return ""
}

func mapPlayOriginID(player map[string]any) string {
	if player == nil {
		return ""
	}
	origin, ok := player["play_origin"].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := origin["device_identifier"].(string)
	return id
}

func mapDevices(raw map[string]any) []domain.Device {
	devices := make([]domain.Device, 0, len(raw))
	for id, item := range raw {
		device, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := device["name"].(string)
		deviceType, _ := device["device_type"].(string)
		isActive, _ := device["is_active"].(bool)
		devices = append(devices, domain.Device{ID: id, Name: name, Type: deviceType, IsActive: isActive})
	}
	return devices
}

func mapPlaybackState(state connectState) *domain.CurrentPlayback {
	playback := &domain.CurrentPlayback{State: domain.PlaybackStateIdle}
	if state.activeDeviceID != "" {
		for _, device := range mapDevices(state.devices) {
			if device.ID == state.activeDeviceID {
				playback.Device = &device
				break
			}
		}
	}
	if state.playerState == nil {
		return playback
	}
	if paused, ok := state.playerState["is_paused"].(bool); ok {
		if paused {
			playback.State = domain.PlaybackStatePaused
		} else {
			playback.State = domain.PlaybackStatePlaying
		}
	}
	track, ok := state.playerState["track"].(map[string]any)
	if !ok {
		return playback
	}
	uri, _ := track["uri"].(string)
	name, _ := track["name"].(string)
	artist := firstArtistName(track["artist"])
	album := nestedString(track, "album", "name")
	if uri != "" || name != "" {
		playback.Track = &domain.Track{URI: uri, Title: name, Artist: artist, Album: album}
	}
	return playback
}

func firstArtistName(raw any) string {
	switch artists := raw.(type) {
	case []any:
		for _, item := range artists {
			if artist, ok := item.(map[string]any); ok {
				if name, _ := artist["name"].(string); name != "" {
					return name
				}
			}
		}
	case map[string]any:
		name, _ := artists["name"].(string)
		return name
	}
	return ""
}

func nestedString(value map[string]any, key, nested string) string {
	raw, ok := value[key].(map[string]any)
	if !ok {
		return ""
	}
	result, _ := raw[nested].(string)
	return result
}
