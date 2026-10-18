package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// fakeSpotify serves the token endpoint and GET /me/top/{type} with canned data.
func fakeSpotify(t *testing.T, fail *atomic.Bool, calls *atomic.Int32) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/token", func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
			return
		}
		if id, secret, ok := r.BasicAuth(); !ok || id != "id" || secret != "secret" {
			t.Errorf("token request basic auth = %q/%q, want id/secret", id, secret)
		}
		if got := r.FormValue("refresh_token"); got != "refresh" {
			t.Errorf("refresh_token = %q, want %q", got, "refresh")
		}
		fmt.Fprint(w, `{"access_token":"access","expires_in":3600}`)
	})
	mux.HandleFunc("GET /v1/me/top/{kind}", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if got := r.Header.Get("Authorization"); got != "Bearer access" {
			t.Errorf("Authorization = %q", got)
		}
		tr := r.URL.Query().Get("time_range")
		switch r.PathValue("kind") {
		case "artists":
			fmt.Fprintf(w, `{"items":[{"name":"Artist %s","external_urls":{"spotify":"https://open.spotify.com/artist/a"},
				"images":[{"url":"https://img/640","width":640},{"url":"https://img/160","width":160},{"url":"https://img/32","width":32}]}]}`, tr)
		case "tracks":
			fmt.Fprintf(w, `{"items":[{"name":"Track %s","external_urls":{"spotify":"https://open.spotify.com/track/t"},
				"artists":[{"name":"A"},{"name":"B"}],"album":{"images":[{"url":"https://img/300","width":300}]}}]}`, tr)
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestSpotifyClientStats(t *testing.T) {
	var fail atomic.Bool
	var calls atomic.Int32
	srv := fakeSpotify(t, &fail, &calls)

	c := &SpotifyClient{
		clientID:     "id",
		clientSecret: "secret",
		refreshToken: "refresh",
		tokenURL:     srv.URL + "/api/token",
		apiBaseURL:   srv.URL + "/v1",
		httpClient:   srv.Client(),
	}

	stats := c.Stats()
	if stats == nil {
		t.Fatal("Stats() = nil, want data")
	}
	if len(stats.Ranges) != len(spotifyRanges) {
		t.Fatalf("got %d ranges, want %d", len(stats.Ranges), len(spotifyRanges))
	}
	for i, r := range stats.Ranges {
		if r.Key != spotifyRanges[i].Key {
			t.Errorf("range %d key = %q, want %q", i, r.Key, spotifyRanges[i].Key)
		}
		want := SpotifyItem{Name: "Artist " + r.Key, URL: "https://open.spotify.com/artist/a", ImageURL: "https://img/160"}
		if len(r.Artists) != 1 || r.Artists[0] != want {
			t.Errorf("range %s artists = %+v, want [%+v]", r.Key, r.Artists, want)
		}
		want = SpotifyItem{Name: "Track " + r.Key, Subtitle: "A, B", URL: "https://open.spotify.com/track/t", ImageURL: "https://img/300"}
		if len(r.Tracks) != 1 || r.Tracks[0] != want {
			t.Errorf("range %s tracks = %+v, want [%+v]", r.Key, r.Tracks, want)
		}
	}
	if got := calls.Load(); got != 6 {
		t.Errorf("API calls = %d, want 6", got)
	}

	// キャッシュが有効な間は Spotify を叩かない。
	if c.Stats() != stats || calls.Load() != 6 {
		t.Error("second Stats() call did not use the cache")
	}

	// キャッシュ切れ後に失敗しても、直前の結果を返し続ける。
	fail.Store(true)
	c.nextFetch = time.Time{}
	c.accessToken = ""
	if got := c.Stats(); got != stats {
		t.Errorf("Stats() after failure = %p, want stale %p", got, stats)
	}
	if !c.nextFetch.After(time.Now()) {
		t.Error("nextFetch was not pushed back after failure")
	}
}

func TestNilSpotifyClient(t *testing.T) {
	var c *SpotifyClient
	if got := c.Stats(); got != nil {
		t.Errorf("nil client Stats() = %+v, want nil", got)
	}
}
