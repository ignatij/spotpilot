// Package spotify implements the SpotifyClient port against the Spotify Web API.
package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ignatij/spotpilot/internal/domain"
)

// Client implements the app.SpotifyClient port.
type Client struct {
	http    *http.Client
	baseURL string
	token   func() (string, error)
}

// New creates a Spotify API client. token is a function that returns the
// current Bearer token, allowing session refresh without recreating the client.
func New(token func() (string, error)) *Client {
	return &Client{
		http:    &http.Client{Timeout: 10 * time.Second},
		baseURL: "https://api.spotify.com/v1",
		token:   token,
	}
}

func (c *Client) authHeader() (string, error) {
	tok, err := c.token()
	if err != nil {
		return "", err
	}
	return "Bearer " + tok, nil
}

func (c *Client) do(ctx context.Context, method, path string, body string) (*http.Response, error) {
	auth, err := c.authHeader()
	if err != nil {
		return nil, err
	}
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	var req *http.Request
	if bodyReader != nil {
		req, err = http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	}
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")
	return c.http.Do(req)
}

// Search searches Spotify for tracks, albums, and artists, returning the
// highest-priority match (track > album > artist).
func (c *Client) Search(ctx context.Context, query string) (*domain.MatchResult, error) {
	resp, err := c.do(ctx, http.MethodGet,
		fmt.Sprintf("/search?q=%s&type=track,album,artist&limit=1", encodeQuery(query)),
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

	if len(sr.Tracks.Items) > 0 {
		t := sr.Tracks.Items[0]
		artist := ""
		if len(t.Artists) > 0 {
			artist = t.Artists[0].Name
		}
		return &domain.MatchResult{
			Type:  domain.MatchTypeTrack,
			Track: &domain.Track{URI: t.URI, Title: t.Name, Artist: artist, Album: t.Album.Name},
		}, nil
	}
	if len(sr.Albums.Items) > 0 {
		a := sr.Albums.Items[0]
		artist := ""
		if len(a.Artists) > 0 {
			artist = a.Artists[0].Name
		}
		return &domain.MatchResult{
			Type:  domain.MatchTypeAlbum,
			Album: &domain.Album{URI: a.URI, Name: a.Name, Artist: artist},
		}, nil
	}
	if len(sr.Artists.Items) > 0 {
		a := sr.Artists.Items[0]
		return &domain.MatchResult{
			Type:   domain.MatchTypeArtist,
			Artist: &domain.Artist{URI: a.URI, Name: a.Name},
		}, nil
	}
	return nil, nil // no match
}

// Play starts playback of uri on deviceID.
func (c *Client) Play(ctx context.Context, deviceID string, uri string) error {
	body := fmt.Sprintf(`{"context_uri":"%s"}`, uri)
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
	return fmt.Errorf("spotify API error: status %d", resp.StatusCode)
}

func encodeQuery(q string) string {
	return strings.ReplaceAll(q, " ", "+")
}

func init() {
	_ = errors.New // ensure errors is used if needed
}
