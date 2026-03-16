// Package spotify implements the SpotifyClient port against the Spotify Web API.
package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ignatij/spotpilot/internal/domain"
)

const searchResultLimit = 5

// Client implements the app.SpotifyClient port.
type Client struct {
	http          *http.Client
	baseURL       string
	tokenProvider func(context.Context) (Token, error)

	mu        sync.Mutex
	lastToken Token
}

// New creates a Spotify API client. token is a function that returns the
// current Bearer token, allowing session refresh without recreating the client.
func New(tokenProvider func(context.Context) (Token, error)) *Client {
	return &Client{
		http:          &http.Client{Timeout: 10 * time.Second},
		baseURL:       "https://api.spotify.com/v1",
		tokenProvider: tokenProvider,
	}
}

func (c *Client) authHeader(ctx context.Context) (string, error) {
	tok, err := c.token(ctx)
	if err != nil {
		return "", err
	}
	return "Bearer " + tok.AccessToken, nil
}

func (c *Client) token(ctx context.Context) (Token, error) {
	c.mu.Lock()
	if c.lastToken.AccessToken != "" && time.Until(c.lastToken.ExpiresAt) > time.Minute {
		tok := c.lastToken
		c.mu.Unlock()
		return tok, nil
	}
	c.mu.Unlock()

	tok, err := c.tokenProvider(ctx)
	if err != nil {
		return Token{}, err
	}

	c.mu.Lock()
	c.lastToken = tok
	c.mu.Unlock()
	return tok, nil
}

func (c *Client) do(ctx context.Context, method, path string, body string) (*http.Response, error) {
	auth, err := c.authHeader(ctx)
	if err != nil {
		return nil, err
	}
	makeRequest := func(reqCtx context.Context) (*http.Response, error) {
		var bodyReader *strings.Reader
		if body != "" {
			bodyReader = strings.NewReader(body)
		}
		var req *http.Request
		if bodyReader != nil {
			req, err = http.NewRequestWithContext(reqCtx, method, c.baseURL+path, bodyReader)
		} else {
			req, err = http.NewRequestWithContext(reqCtx, method, c.baseURL+path, nil)
		}
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", auth)
		req.Header.Set("Content-Type", "application/json")
		return c.http.Do(req)
	}

	resp, err := makeRequest(ctx)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		return resp, nil
	}

	retryAfter, ok := retryAfterDuration(resp)
	if !ok || retryAfter <= 0 || retryAfter > 5*time.Second {
		return resp, nil
	}
	resp.Body.Close()

	timer := time.NewTimer(retryAfter)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
	}

	return makeRequest(ctx)
}

// Search searches Spotify for tracks, albums, and artists, returning the
// highest-priority match (track > album > artist).
func (c *Client) Search(ctx context.Context, query string) (*domain.MatchResult, error) {
	resp, err := c.do(ctx, http.MethodGet,
		fmt.Sprintf("/search?q=%s&type=track,album,artist&limit=%d", encodeQuery(query), searchResultLimit),
		"")
	if err != nil {
		return nil, fmt.Errorf("spotify search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp)
	}

	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("decoding search response: %w", err)
	}

	match := matchResultFromSearchResponse(query, sr)
	if match == nil {
		return nil, nil
	}
	return c.hydrateMatch(ctx, match), nil
}

func matchResultFromSearchResponse(query string, sr searchResponse) *domain.MatchResult {
	return bestSearchMatch(query, sr.Tracks.Items, sr.Albums.Items, sr.Artists.Items)
}

func bestSearchMatch(query string, tracks []trackItem, albums []albumItem, artists []artistItem) *domain.MatchResult {
	type candidate struct {
		score int
		match *domain.MatchResult
	}

	queryIntent := searchIntent(query)
	queryText := canonicalSearchQuery(query)
	best := candidate{score: -1}

	for idx, item := range tracks {
		artist := firstArtistItemName(item.Artists)
		match := &domain.MatchResult{
			Type:  domain.MatchTypeTrack,
			Track: &domain.Track{URI: item.URI, Title: item.Name, Artist: artist, Album: item.Album.Name},
		}
		score := searchCandidateScore(queryText, queryIntent, domain.MatchTypeTrack, item.Name, artist, idx)
		if score > best.score {
			best = candidate{score: score, match: match}
		}
	}

	for idx, item := range albums {
		artist := firstArtistItemName(item.Artists)
		match := &domain.MatchResult{
			Type:  domain.MatchTypeAlbum,
			Album: &domain.Album{URI: item.URI, Name: item.Name, Artist: artist},
		}
		score := searchCandidateScore(queryText, queryIntent, domain.MatchTypeAlbum, item.Name, artist, idx)
		if score > best.score {
			best = candidate{score: score, match: match}
		}
	}

	for idx, item := range artists {
		match := &domain.MatchResult{
			Type:   domain.MatchTypeArtist,
			Artist: &domain.Artist{URI: item.URI, Name: item.Name},
		}
		score := searchCandidateScore(queryText, queryIntent, domain.MatchTypeArtist, item.Name, "", idx)
		if score > best.score {
			best = candidate{score: score, match: match}
		}
	}

	return best.match
}

func searchCandidateScore(queryText string, intent domain.MatchType, kind domain.MatchType, name string, secondary string, index int) int {
	nameScore := matchTextScore(queryText, name)
	secondaryScore := matchSecondaryScore(queryText, secondary)
	typeScore := 0
	if kind == domain.MatchTypeTrack {
		typeScore = 150
	} else if kind == domain.MatchTypeAlbum {
		typeScore = 75
	}
	intentScore := 0
	if intent != "" && intent == kind {
		intentScore = 200
	}
	return nameScore + secondaryScore + typeScore - index + intentScore
}

func matchTextScore(query string, value string) int {
	query = normalizeSearchText(query)
	value = normalizeSearchText(value)
	if query == "" || value == "" {
		return 0
	}
	if value == query {
		return 1000
	}
	if strings.HasPrefix(value, query+" ") || strings.HasPrefix(value, query) {
		return 850
	}
	if strings.Contains(value, query) {
		return 700
	}
	queryTokens := strings.Fields(query)
	valueTokens := strings.Fields(value)
	overlap := overlappingTokenCount(queryTokens, valueTokens)
	if overlap == len(queryTokens) && overlap > 0 {
		return 500
	}
	if overlap > 0 {
		return overlap * 100
	}
	return 0
}

func matchSecondaryScore(query string, value string) int {
	query = normalizeSearchText(query)
	value = normalizeSearchText(value)
	if query == "" || value == "" {
		return 0
	}
	if value == query {
		return 60
	}
	if strings.Contains(value, query) {
		return 40
	}
	queryTokens := strings.Fields(query)
	valueTokens := strings.Fields(value)
	overlap := overlappingTokenCount(queryTokens, valueTokens)
	if overlap > 0 {
		return overlap * 20
	}
	return 0
}

func overlappingTokenCount(left []string, right []string) int {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	seen := make(map[string]struct{}, len(right))
	for _, token := range right {
		seen[token] = struct{}{}
	}
	count := 0
	for _, token := range left {
		if _, ok := seen[token]; ok {
			count++
		}
	}
	return count
}

func searchIntent(query string) domain.MatchType {
	trimmed := strings.TrimSpace(strings.ToLower(query))
	switch {
	case strings.HasPrefix(trimmed, "track:"), strings.HasPrefix(trimmed, "track "), strings.HasSuffix(trimmed, " track"), strings.HasSuffix(trimmed, " song"):
		return domain.MatchTypeTrack
	case strings.HasPrefix(trimmed, "album:"), strings.HasPrefix(trimmed, "album "), strings.HasSuffix(trimmed, " album"):
		return domain.MatchTypeAlbum
	case strings.HasPrefix(trimmed, "artist:"), strings.HasPrefix(trimmed, "artist "), strings.HasSuffix(trimmed, " artist"):
		return domain.MatchTypeArtist
	default:
		return ""
	}
}

func canonicalSearchQuery(query string) string {
	trimmed := strings.TrimSpace(query)
	lower := strings.ToLower(trimmed)
	for _, prefix := range []string{"track:", "track ", "song ", "album:", "album ", "artist:", "artist "} {
		if strings.HasPrefix(lower, prefix) {
			return strings.TrimSpace(trimmed[len(prefix):])
		}
	}
	for _, suffix := range []string{" track", " song", " album", " artist"} {
		if strings.HasSuffix(lower, suffix) {
			return strings.TrimSpace(trimmed[:len(trimmed)-len(suffix)])
		}
	}
	return trimmed
}

func normalizeSearchText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == ' ':
			return r
		default:
			return ' '
		}
	}, value)
	return strings.Join(strings.Fields(value), " ")
}

func firstArtistItemName(artists []artistItem) string {
	if len(artists) == 0 {
		return ""
	}
	return artists[0].Name
}

func (c *Client) hydrateMatch(ctx context.Context, match *domain.MatchResult) *domain.MatchResult {
	if match == nil {
		return nil
	}
	if !matchNeedsHydration(match) {
		return match
	}
	kind, id, ok := spotifyURITypeAndID(match.PlaybackURI())
	if !ok {
		return match
	}

	switch kind {
	case "track":
		if hydrated, err := c.getTrack(ctx, id); err == nil && hydrated != nil {
			return &domain.MatchResult{Type: domain.MatchTypeTrack, Track: hydrated}
		}
	case "album":
		if hydrated, err := c.getAlbum(ctx, id); err == nil && hydrated != nil {
			return &domain.MatchResult{Type: domain.MatchTypeAlbum, Album: hydrated}
		}
	case "artist":
		if hydrated, err := c.getArtist(ctx, id); err == nil && hydrated != nil {
			return &domain.MatchResult{Type: domain.MatchTypeArtist, Artist: hydrated}
		}
	}
	return match
}

func matchNeedsHydration(match *domain.MatchResult) bool {
	if match == nil {
		return false
	}
	switch match.Type {
	case domain.MatchTypeTrack:
		if match.Track == nil {
			return true
		}
		return strings.TrimSpace(match.Track.URI) == "" || strings.TrimSpace(match.Track.Title) == "" || strings.TrimSpace(match.Track.Artist) == ""
	case domain.MatchTypeAlbum:
		if match.Album == nil {
			return true
		}
		return strings.TrimSpace(match.Album.URI) == "" || strings.TrimSpace(match.Album.Name) == "" || strings.TrimSpace(match.Album.Artist) == ""
	case domain.MatchTypeArtist:
		if match.Artist == nil {
			return true
		}
		return strings.TrimSpace(match.Artist.URI) == "" || strings.TrimSpace(match.Artist.Name) == ""
	default:
		return false
	}
}

func spotifyURITypeAndID(uri string) (kind string, id string, ok bool) {
	parts := strings.Split(strings.TrimSpace(uri), ":")
	if len(parts) < 3 || !strings.EqualFold(parts[0], "spotify") {
		return "", "", false
	}
	kind = strings.TrimSpace(parts[1])
	id = strings.TrimSpace(parts[2])
	if kind == "" || id == "" {
		return "", "", false
	}
	return kind, id, true
}

func (c *Client) getTrack(ctx context.Context, id string) (*domain.Track, error) {
	resp, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/tracks/%s", id), "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp)
	}
	var item trackItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, err
	}
	artist := ""
	if len(item.Artists) > 0 {
		artist = item.Artists[0].Name
	}
	return &domain.Track{URI: item.URI, Title: item.Name, Artist: artist, Album: item.Album.Name}, nil
}

func (c *Client) getAlbum(ctx context.Context, id string) (*domain.Album, error) {
	resp, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/albums/%s", id), "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp)
	}
	var item albumItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, err
	}
	artist := ""
	if len(item.Artists) > 0 {
		artist = item.Artists[0].Name
	}
	return &domain.Album{URI: item.URI, Name: item.Name, Artist: artist}, nil
}

func (c *Client) getArtist(ctx context.Context, id string) (*domain.Artist, error) {
	resp, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/artists/%s", id), "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp)
	}
	var item artistItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, err
	}
	return &domain.Artist{URI: item.URI, Name: item.Name}, nil
}

// Play starts playback of uri on deviceID.
func (c *Client) Play(ctx context.Context, deviceID string, uri string) error {
	body := fmt.Sprintf(`{"context_uri":"%s"}`, uri)
	if strings.HasPrefix(strings.TrimSpace(uri), "spotify:track:") {
		body = fmt.Sprintf(`{"uris":[%q]}`, strings.TrimSpace(uri))
	}
	resp, err := c.do(ctx, http.MethodPut,
		fmt.Sprintf("/me/player/play?device_id=%s", deviceID), body)
	if err != nil {
		return fmt.Errorf("spotify play: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return apiError(resp)
	}
	return nil
}

// Pause pauses playback on deviceID.
func (c *Client) Pause(ctx context.Context, deviceID string) error {
	resp, err := c.do(ctx, http.MethodPut,
		fmt.Sprintf("/me/player/pause?device_id=%s", deviceID), "")
	if err != nil {
		return fmt.Errorf("spotify pause: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return apiError(resp)
	}
	return nil
}

// Resume resumes playback on deviceID.
func (c *Client) Resume(ctx context.Context, deviceID string) error {
	resp, err := c.do(ctx, http.MethodPut,
		fmt.Sprintf("/me/player/play?device_id=%s", deviceID), "")
	if err != nil {
		return fmt.Errorf("spotify resume: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return apiError(resp)
	}
	return nil
}

// Next skips to the next track.
func (c *Client) Next(ctx context.Context, deviceID string) error {
	resp, err := c.do(ctx, http.MethodPost,
		fmt.Sprintf("/me/player/next?device_id=%s", deviceID), "")
	if err != nil {
		return fmt.Errorf("spotify next: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return apiError(resp)
	}
	return nil
}

// Previous skips to the previous track.
func (c *Client) Previous(ctx context.Context, deviceID string) error {
	resp, err := c.do(ctx, http.MethodPost,
		fmt.Sprintf("/me/player/previous?device_id=%s", deviceID), "")
	if err != nil {
		return fmt.Errorf("spotify previous: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return apiError(resp)
	}
	return nil
}

// CurrentPlayback returns the current playback state.
func (c *Client) CurrentPlayback(ctx context.Context) (*domain.CurrentPlayback, error) {
	resp, err := c.do(ctx, http.MethodGet, "/me/player", "")
	if err != nil {
		return nil, fmt.Errorf("spotify current playback: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		// No active player.
		return &domain.CurrentPlayback{State: domain.PlaybackStateIdle}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp)
	}

	var pr playerResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("decoding player response: %w", err)
	}

	pb := &domain.CurrentPlayback{}
	if pr.IsPlaying {
		pb.State = domain.PlaybackStatePlaying
	} else {
		pb.State = domain.PlaybackStatePaused
	}
	if pr.Device.ID != "" {
		pb.Device = &domain.Device{
			ID:   pr.Device.ID,
			Name: pr.Device.Name,
			Type: pr.Device.Type,
		}
	}
	if pr.Item.URI != "" {
		artist := ""
		if len(pr.Item.Artists) > 0 {
			artist = pr.Item.Artists[0].Name
		}
		pb.Track = &domain.Track{
			URI:    pr.Item.URI,
			Title:  pr.Item.Name,
			Artist: artist,
			Album:  pr.Item.Album.Name,
		}
	}
	return pb, nil
}

// ListDevices returns available Spotify devices.
func (c *Client) ListDevices(ctx context.Context) ([]domain.Device, error) {
	resp, err := c.do(ctx, http.MethodGet, "/me/player/devices", "")
	if err != nil {
		return nil, fmt.Errorf("spotify list devices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp)
	}

	var dr devicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		return nil, fmt.Errorf("decoding devices response: %w", err)
	}

	devices := make([]domain.Device, 0, len(dr.Devices))
	for _, d := range dr.Devices {
		devices = append(devices, domain.Device{
			ID:       d.ID,
			Name:     d.Name,
			Type:     d.Type,
			IsActive: d.IsActive,
		})
	}
	return devices, nil
}

func apiError(resp *http.Response) error {
	if resp.StatusCode == http.StatusTooManyRequests {
		if retryAfter, ok := retryAfterDuration(resp); ok {
			return fmt.Errorf("spotify API error: status %d retry_after=%s", resp.StatusCode, retryAfter)
		}
	}
	return fmt.Errorf("spotify API error: status %d", resp.StatusCode)
}

func retryAfterDuration(resp *http.Response) (time.Duration, bool) {
	value := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if value == "" {
		return 0, false
	}
	seconds, err := time.ParseDuration(value + "s")
	if err == nil {
		return seconds, true
	}
	return 0, false
}

func encodeQuery(q string) string {
	return strings.ReplaceAll(q, " ", "+")
}

func (c *Client) transferPlayback(ctx context.Context, deviceID string) error {
	if strings.TrimSpace(deviceID) == "" {
		return nil
	}
	body := fmt.Sprintf(`{"device_ids":[%q],"play":false}`, deviceID)
	resp, err := c.do(ctx, http.MethodPut, "/me/player", body)
	if err != nil {
		return fmt.Errorf("spotify transfer playback: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return apiError(resp)
	}
	return nil
}

func init() {
	_ = errors.New // ensure errors is used if needed
}
