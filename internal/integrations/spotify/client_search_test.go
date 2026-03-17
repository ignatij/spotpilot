package spotify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientSearchPrefersRelevantTrackOverArtist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/search" {
			t.Fatalf("unexpected path: %s", got)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Fatalf("unexpected limit: %s", got)
		}
		_ = json.NewEncoder(w).Encode(searchResponse{
			Tracks: trackPage{Items: []trackItem{
				{URI: "spotify:track:enter-sandman", Name: "Enter Sandman", Artists: []artistItem{{Name: "Metallica"}}, Album: struct {
					Name string `json:"name"`
				}{Name: "Metallica"}},
				{URI: "spotify:track:battery", Name: "Battery", Artists: []artistItem{{Name: "Metallica"}}, Album: struct {
					Name string `json:"name"`
				}{Name: "Master of Puppets"}},
			}},
			Albums:  albumPage{Items: []albumItem{{URI: "spotify:album:battery", Name: "Battery", Artists: []artistItem{{Name: "Battery"}}}}},
			Artists: artistPage{Items: []artistItem{{URI: "spotify:artist:battery", Name: "Battery"}}},
		})
	}))
	defer server.Close()

	client := New(func(context.Context) (Token, error) {
		return Token{AccessToken: "token", ExpiresAt: time.Now().Add(time.Hour)}, nil
	})
	client.baseURL = server.URL

	match, err := client.Search(context.Background(), "Battery")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if match == nil {
		t.Fatal("expected match")
		return
	}
	if match.Type != "track" {
		t.Fatalf("expected track match, got %s", match.Type)
	}
	if match.Track == nil || match.Track.URI != "spotify:track:battery" {
		t.Fatalf("unexpected track match: %#v", match.Track)
	}
	if match.Track.Title != "Battery" {
		t.Fatalf("unexpected title: %q", match.Track.Title)
	}
}
