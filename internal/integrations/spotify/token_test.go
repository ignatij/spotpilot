package spotify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type stubCookieSource struct{}

func (stubCookieSource) Cookies(context.Context) ([]*http.Cookie, error) {
	return []*http.Cookie{{Name: "sp_dc", Value: "cookie", Path: "/"}}, nil
}

func TestCookieTokenProvider(t *testing.T) {
	secretPath := filepath.Join(t.TempDir(), "secret.json")
	secretData, err := json.Marshal(map[string][]int{"1": {1, 2, 3, 4}})
	if err != nil {
		t.Fatalf("marshal secret: %v", err)
	}
	if err := os.WriteFile(secretPath, secretData, 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	t.Setenv(totpSecretEnv, secretPath)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if got := r.URL.Query().Get("reason"); got != "init" {
			t.Fatalf("unexpected reason: %q", got)
		}
		if got := r.URL.Query().Get("productType"); got != "web-player" {
			t.Fatalf("unexpected productType: %q", got)
		}
		if r.URL.Query().Get("totp") == "" || r.URL.Query().Get("totpVer") == "" || r.URL.Query().Get("totpServer") == "" {
			t.Fatalf("expected totp query params, got %q", r.URL.RawQuery)
		}
		if _, err := r.Cookie("sp_dc"); err != nil {
			t.Fatalf("expected sp_dc cookie: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accessToken":                      "token-123",
			"accessTokenExpirationTimestampMs": time.Now().Add(30 * time.Minute).UnixMilli(),
			"isAnonymous":                      false,
			"clientId":                         "client-123",
		})
	}))
	defer server.Close()

	provider := CookieTokenProvider{
		Source:  stubCookieSource{},
		BaseURL: server.URL + "/",
	}

	tok, err := provider.Token(context.Background())
	if err != nil {
		t.Fatalf("provider token: %v", err)
	}
	if tok.AccessToken != "token-123" {
		t.Fatalf("unexpected token: %q", tok.AccessToken)
	}
	if tok.ClientID != "client-123" {
		t.Fatalf("unexpected client id: %q", tok.ClientID)
	}
	if time.Until(tok.ExpiresAt) <= 0 {
		t.Fatal("expected token expiry in the future")
	}
}
