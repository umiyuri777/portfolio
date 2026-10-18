package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Spotify Web API の「Get User's Top Items」(GET /me/top/{type}) を使って、
// 期間ごとによく聴いているアーティスト・曲を取得する。
// API は再生回数を返さないため、ページに出せるのは順位のみ。
const (
	spotifyTokenURL   = "https://accounts.spotify.com/api/token"
	spotifyAPIBaseURL = "https://api.spotify.com/v1"

	spotifyTopLimit = 10

	// トップアイテムは頻繁には変わらないので、Spotify へのアクセスは1時間に1回で十分。
	spotifyCacheTTL = time.Hour
	// 失敗時に毎リクエスト Spotify を叩かないよう、再試行まで間隔を空ける。
	spotifyRetryInterval = 5 * time.Minute
	spotifyFetchTimeout  = 5 * time.Second
)

// spotifyRanges maps Spotify's time_range values to the labels shown on the page.
// short_term ≒ 直近4週間, medium_term ≒ 直近6ヶ月, long_term ≒ 直近1年。
var spotifyRanges = []struct{ Key, Label string }{
	{Key: "short_term", Label: "1ヶ月"},
	{Key: "medium_term", Label: "半年"},
	{Key: "long_term", Label: "1年"},
}

// SpotifyStats is the per-range ranking rendered in the Spotify section.
type SpotifyStats struct {
	Ranges    []SpotifyRange
	FetchedAt time.Time
}

// SpotifyRange holds the top artists and tracks for one time_range.
type SpotifyRange struct {
	Key     string
	Label   string
	Artists []SpotifyItem
	Tracks  []SpotifyItem
}

// SpotifyItem is a single artist or track. For tracks, Subtitle holds the artist names.
type SpotifyItem struct {
	Name     string
	Subtitle string
	URL      string
	ImageURL string
}

// SpotifyClient fetches and caches the owner's top items using a long-lived refresh token.
type SpotifyClient struct {
	clientID     string
	clientSecret string
	tokenURL     string
	apiBaseURL   string
	httpClient   *http.Client

	// mu はトークンとキャッシュをまとめて守る。取得中も保持し続けることで、
	// キャッシュ切れの瞬間に同時リクエストが Spotify を多重に叩かないようにしている。
	mu           sync.Mutex
	refreshToken string
	accessToken  string
	tokenExpiry  time.Time
	stats        *SpotifyStats
	nextFetch    time.Time
}

// NewSpotifyClientFromEnv returns nil when credentials are not configured,
// so the page (and local development) works without a Spotify app.
func NewSpotifyClientFromEnv() *SpotifyClient {
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	refreshToken := os.Getenv("SPOTIFY_REFRESH_TOKEN")
	if clientID == "" || clientSecret == "" || refreshToken == "" {
		return nil
	}
	return &SpotifyClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		refreshToken: refreshToken,
		tokenURL:     spotifyTokenURL,
		apiBaseURL:   spotifyAPIBaseURL,
		httpClient:   &http.Client{Timeout: spotifyFetchTimeout},
	}
}

// Stats returns the cached stats, refreshing them once the cache has expired.
// 取得に失敗しても直前の成功結果（なければ nil）を返し、ページ自体は必ず描画できるようにする。
func (s *SpotifyClient) Stats() *SpotifyStats {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if time.Now().Before(s.nextFetch) {
		return s.stats
	}

	// リクエストの context は使わない: 閲覧者が途中で離脱しても取得は最後まで終わらせたい。
	ctx, cancel := context.WithTimeout(context.Background(), spotifyFetchTimeout)
	defer cancel()

	stats, err := s.fetch(ctx)
	if err != nil {
		log.Printf("spotify: %v", err)
		s.nextFetch = time.Now().Add(spotifyRetryInterval)
		return s.stats
	}
	s.stats = stats
	s.nextFetch = time.Now().Add(spotifyCacheTTL)
	return stats
}

func (s *SpotifyClient) fetch(ctx context.Context) (*SpotifyStats, error) {
	token, err := s.token(ctx)
	if err != nil {
		return nil, err
	}

	stats := &SpotifyStats{
		Ranges:    make([]SpotifyRange, len(spotifyRanges)),
		FetchedAt: time.Now(),
	}
	// 3期間 × (artists, tracks) の6リクエストを並列に投げる。
	errs := make([]error, 2*len(spotifyRanges))
	var wg sync.WaitGroup
	for i, rng := range spotifyRanges {
		r := &stats.Ranges[i]
		r.Key, r.Label = rng.Key, rng.Label

		wg.Add(2)
		go func() {
			defer wg.Done()
			r.Artists, errs[2*i] = s.topArtists(ctx, token, rng.Key)
		}()
		go func() {
			defer wg.Done()
			r.Tracks, errs[2*i+1] = s.topTracks(ctx, token, rng.Key)
		}()
	}
	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return stats, nil
}

// token returns a valid access token, exchanging the refresh token when needed.
func (s *SpotifyClient) token(ctx context.Context) (string, error) {
	if s.accessToken != "" && time.Now().Before(s.tokenExpiry) {
		return s.accessToken, nil
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {s.refreshToken},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.clientID, s.clientSecret)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		// リフレッシュトークンは認可から6ヶ月で失効する（invalid_grant）。
		// その場合は cmd/spotify-auth で再認可して SPOTIFY_REFRESH_TOKEN を差し替える。
		return "", fmt.Errorf("refresh token: %s: %s", resp.Status, body)
	}

	var tok struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", fmt.Errorf("refresh token: decode: %w", err)
	}

	s.accessToken = tok.AccessToken
	// 期限ぎりぎりで失効しないよう1分の余裕を持たせる。
	s.tokenExpiry = time.Now().Add(time.Duration(tok.ExpiresIn)*time.Second - time.Minute)
	// 新しいリフレッシュトークンが返ってきた場合はそちらを使い続ける。
	if tok.RefreshToken != "" {
		s.refreshToken = tok.RefreshToken
	}
	return s.accessToken, nil
}

type spotifyImage struct {
	URL   string `json:"url"`
	Width int    `json:"width"`
}

type spotifyExternalURLs struct {
	Spotify string `json:"spotify"`
}

type spotifyArtist struct {
	Name         string              `json:"name"`
	ExternalURLs spotifyExternalURLs `json:"external_urls"`
	Images       []spotifyImage      `json:"images"`
}

type spotifyTrack struct {
	Name         string              `json:"name"`
	ExternalURLs spotifyExternalURLs `json:"external_urls"`
	Artists      []spotifyArtist     `json:"artists"`
	Album        struct {
		Images []spotifyImage `json:"images"`
	} `json:"album"`
}

func (s *SpotifyClient) topArtists(ctx context.Context, token, timeRange string) ([]SpotifyItem, error) {
	artists, err := getTopItems[spotifyArtist](ctx, s, token, "artists", timeRange)
	if err != nil {
		return nil, err
	}
	items := make([]SpotifyItem, 0, len(artists))
	for _, a := range artists {
		items = append(items, SpotifyItem{
			Name:     a.Name,
			URL:      a.ExternalURLs.Spotify,
			ImageURL: thumbnailURL(a.Images),
		})
	}
	return items, nil
}

func (s *SpotifyClient) topTracks(ctx context.Context, token, timeRange string) ([]SpotifyItem, error) {
	tracks, err := getTopItems[spotifyTrack](ctx, s, token, "tracks", timeRange)
	if err != nil {
		return nil, err
	}
	items := make([]SpotifyItem, 0, len(tracks))
	for _, t := range tracks {
		names := make([]string, 0, len(t.Artists))
		for _, a := range t.Artists {
			names = append(names, a.Name)
		}
		items = append(items, SpotifyItem{
			Name:     t.Name,
			Subtitle: strings.Join(names, ", "),
			URL:      t.ExternalURLs.Spotify,
			ImageURL: thumbnailURL(t.Album.Images),
		})
	}
	return items, nil
}

// getTopItems calls GET /me/top/{kind}. Go のメソッドは型パラメータを持てないので関数にしている。
func getTopItems[T any](ctx context.Context, s *SpotifyClient, token, kind, timeRange string) ([]T, error) {
	q := url.Values{
		"time_range": {timeRange},
		"limit":      {fmt.Sprint(spotifyTopLimit)},
	}
	endpoint := s.apiBaseURL + "/me/top/" + kind + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("top %s (%s): %w", kind, timeRange, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("top %s (%s): %s: %s", kind, timeRange, resp.Status, body)
	}

	var page struct {
		Items []T `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("top %s (%s): decode: %w", kind, timeRange, err)
	}
	return page.Items, nil
}

// thumbnailURL picks the smallest image that is still sharp at the rendered size.
// Spotify は images を大きい順に返す。
func thumbnailURL(images []spotifyImage) string {
	const minWidth = 64
	best := ""
	for _, img := range images {
		if best == "" || img.Width >= minWidth {
			best = img.URL
		}
	}
	return best
}
