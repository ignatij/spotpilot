package spotify

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultTokenBaseURL = "https://open.spotify.com/"
	totpSecretEnv       = "SPOTPILOT_TOTP_SECRET_URL"
	totpCacheTTL        = 15 * time.Minute
	fallbackTotpVer     = 18
	totpDigits          = 6
	totpStepSeconds     = 30
	totpHTTPTimeout     = 5 * time.Second
)

var totpSecretURLs = []string{
	"https://github.com/xyloflake/spot-secrets-go/blob/main/secrets/secretDict.json?raw=true",
	"https://github.com/Thereallo1026/spotify-secrets/blob/main/secrets/secretDict.json?raw=true",
	"https://code.thetadev.de/ThetaDev/spotify-secrets/raw/branch/main/secrets/secretDict.json",
}

var fallbackTotpSecret = []byte{70, 60, 33, 57, 92, 120, 90, 33, 32, 62, 62, 55, 126, 93, 66, 35, 108, 68}

type Token struct {
	AccessToken string
	ExpiresAt   time.Time
	Anonymous   bool
	ClientID    string
}

type CookieSource interface {
	Cookies(ctx context.Context) ([]*http.Cookie, error)
}

type CookieSourceFunc func(context.Context) ([]*http.Cookie, error)

func (f CookieSourceFunc) Cookies(ctx context.Context) ([]*http.Cookie, error) {
	return f(ctx)
}

type CookieTokenProvider struct {
	Source  CookieSource
	BaseURL string
	Client  *http.Client
}

type accessTokenResponse struct {
	AccessToken                      string `json:"accessToken"`
	ExpiresIn                        int    `json:"expiresIn"`
	AccessTokenExpirationTimestampMs int64  `json:"accessTokenExpirationTimestampMs"`
	IsAnonymous                      bool   `json:"isAnonymous"`
	ClientID                         string `json:"clientId"`
}

type totpCache struct {
	mu      sync.Mutex
	version int
	secret  []byte
	expires time.Time
}

var cachedTotp totpCache

func (p CookieTokenProvider) Token(ctx context.Context) (Token, error) {
	if p.Source == nil {
		return Token{}, errors.New("cookie source required")
	}
	cookiesList, err := p.Source.Cookies(ctx)
	if err != nil {
		return Token{}, err
	}
	if !containsCookie(cookiesList, "sp_dc") {
		return Token{}, errors.New("missing sp_dc cookie; run 'spotpilot login' again")
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return Token{}, err
	}
	base := strings.TrimSpace(p.BaseURL)
	if base == "" {
		base = defaultTokenBaseURL
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	baseURL, _ := url.Parse(base)
	jar.SetCookies(baseURL, cookiesList)

	client := p.Client
	if client == nil {
		client = &http.Client{Jar: jar, Timeout: 10 * time.Second}
	} else {
		client.Jar = jar
	}

	code, version, err := generateTOTP(ctx, time.Now())
	if err != nil {
		return Token{}, err
	}

	params := url.Values{}
	params.Set("reason", "init")
	params.Set("productType", "web-player")
	params.Set("totp", code)
	params.Set("totpVer", strconv.Itoa(version))
	params.Set("totpServer", code)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"api/token?"+params.Encode(), nil)
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("app-platform", "WebPlayer")
	req.Header.Set("Origin", "https://open.spotify.com")
	req.Header.Set("Referer", "https://open.spotify.com/")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-CH-UA", `"Chromium";v="131", "Not_A Brand";v="24", "Google Chrome";v="131"`)
	req.Header.Set("Sec-CH-UA-Platform", `"macOS"`)
	req.Header.Set("Sec-CH-UA-Mobile", "?0")

	resp, err := client.Do(req)
	if err != nil {
		return Token{}, fmt.Errorf("fetching cookie-backed token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return Token{}, fmt.Errorf("cookie-backed token request failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload accessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Token{}, err
	}
	if payload.AccessToken == "" || payload.IsAnonymous {
		return Token{}, errors.New("missing access token from Spotify; run 'spotpilot login' again")
	}

	expiresAt := time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	if payload.AccessTokenExpirationTimestampMs > 0 {
		expiresAt = time.UnixMilli(payload.AccessTokenExpirationTimestampMs)
	}

	return Token{
		AccessToken: payload.AccessToken,
		ExpiresAt:   expiresAt,
		Anonymous:   payload.IsAnonymous,
		ClientID:    payload.ClientID,
	}, nil
}

func containsCookie(cookies []*http.Cookie, name string) bool {
	for _, cookie := range cookies {
		if cookie.Name == name && cookie.Value != "" {
			return true
		}
	}
	return false
}

func generateTOTP(ctx context.Context, now time.Time) (string, int, error) {
	version, secret := totpSecret(ctx)
	code, err := totpFromSecret(secret, now)
	return code, version, err
}

func totpSecret(ctx context.Context) (int, []byte) {
	now := time.Now()
	cachedTotp.mu.Lock()
	if now.Before(cachedTotp.expires) && len(cachedTotp.secret) > 0 {
		version := cachedTotp.version
		secret := append([]byte(nil), cachedTotp.secret...)
		cachedTotp.mu.Unlock()
		return version, secret
	}
	cachedTotp.mu.Unlock()

	version, secret, err := fetchTotpSecret(ctx)
	if err != nil || len(secret) == 0 {
		return fallbackTotpVer, append([]byte(nil), fallbackTotpSecret...)
	}

	cachedTotp.mu.Lock()
	cachedTotp.version = version
	cachedTotp.secret = append([]byte(nil), secret...)
	cachedTotp.expires = now.Add(totpCacheTTL)
	cachedTotp.mu.Unlock()

	return version, secret
}

func fetchTotpSecret(ctx context.Context) (int, []byte, error) {
	var sources []string
	if override := strings.TrimSpace(os.Getenv(totpSecretEnv)); override != "" {
		sources = []string{override}
	} else {
		sources = append([]string(nil), totpSecretURLs...)
	}

	var lastErr error
	for _, source := range sources {
		version, secret, err := fetchTotpSecretSource(ctx, source)
		if err == nil && len(secret) > 0 {
			return version, secret, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = errors.New("totp secrets missing")
	}
	return 0, nil, lastErr
}

func fetchTotpSecretSource(ctx context.Context, source string) (int, []byte, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		client := &http.Client{Timeout: totpHTTPTimeout}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return 0, nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return 0, nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return 0, nil, fmt.Errorf("totp secrets status %d", resp.StatusCode)
		}
		return parseTotpSecret(resp.Body)
	}

	path := strings.TrimPrefix(source, "file://")
	file, err := os.Open(path)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = file.Close() }()
	return parseTotpSecret(file)
}

func parseTotpSecret(reader io.Reader) (int, []byte, error) {
	var raw map[string][]int
	if err := json.NewDecoder(reader).Decode(&raw); err != nil {
		return 0, nil, err
	}

	bestVersion := -1
	var bestSecret []int
	for key, values := range raw {
		version, err := strconv.Atoi(key)
		if err != nil {
			continue
		}
		if version > bestVersion {
			bestVersion = version
			bestSecret = values
		}
	}
	if bestVersion < 0 || len(bestSecret) == 0 {
		return 0, nil, errors.New("totp secrets missing")
	}

	secret := make([]byte, len(bestSecret))
	for index, value := range bestSecret {
		if value < 0 || value > 255 {
			return 0, nil, errors.New("totp secret out of range")
		}
		secret[index] = byte(value)
	}
	return bestVersion, secret, nil
}

func totpFromSecret(secret []byte, now time.Time) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("totp secret empty")
	}

	transformed := make([]byte, len(secret))
	for index, value := range secret {
		transformed[index] = value ^ byte((index%33)+9)
	}

	var builder strings.Builder
	builder.Grow(len(transformed) * 3)
	for _, value := range transformed {
		builder.WriteString(strconv.Itoa(int(value)))
	}
	return totp([]byte(builder.String()), now), nil
}

func totp(key []byte, now time.Time) string {
	counter := uint64(now.Unix() / totpStepSeconds)
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], counter)

	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(message[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	binCode := (uint32(sum[offset])&0x7f)<<24 |
		(uint32(sum[offset+1])&0xff)<<16 |
		(uint32(sum[offset+2])&0xff)<<8 |
		(uint32(sum[offset+3]) & 0xff)
	code := int(binCode % 1000000)
	return fmt.Sprintf("%0*d", totpDigits, code)
}
