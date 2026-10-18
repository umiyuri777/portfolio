// Command spotify-auth issues the refresh token the portfolio server uses to read
// the owner's Spotify top artists / tracks.
//
// Spotify のリフレッシュトークンは認可から6ヶ月で失効するため、
// 期限が近づいたら（またはサーバーログに invalid_grant が出たら）再実行して差し替える。
//
//	SPOTIFY_CLIENT_ID=... SPOTIFY_CLIENT_SECRET=... go run ./cmd/spotify-auth
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	authorizeURL = "https://accounts.spotify.com/authorize"
	tokenURL     = "https://accounts.spotify.com/api/token"
	// サーバー側で必要なのは GET /me/top/{type} だけなので、スコープも最小限にしている。
	scope = "user-top-read"
)

func main() {
	// Spotify は redirect URI に localhost を許可しておらず、ループバックは IP 表記が必須。
	redirectURI := flag.String("redirect-uri", "http://127.0.0.1:8888/callback",
		"Spotify アプリに登録した redirect URI（ループバックIPのみ対応）")
	flag.Parse()

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatal("SPOTIFY_CLIENT_ID と SPOTIFY_CLIENT_SECRET を環境変数で指定してください")
	}

	redirect, err := url.Parse(*redirectURI)
	if err != nil {
		log.Fatalf("invalid redirect URI: %v", err)
	}

	state := randomState()
	authURL := authorizeURL + "?" + url.Values{
		"client_id":     {clientID},
		"response_type": {"code"},
		"redirect_uri":  {*redirectURI},
		"scope":         {scope},
		"state":         {state},
	}.Encode()

	ln, err := net.Listen("tcp", redirect.Host)
	if err != nil {
		log.Fatalf("listen %s: %v", redirect.Host, err)
	}

	result := make(chan error, 1)
	mux := http.NewServeMux()
	mux.HandleFunc(redirect.Path, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		if e := q.Get("error"); e != "" {
			http.Error(w, "authorization failed: "+e, http.StatusBadRequest)
			result <- fmt.Errorf("authorization failed: %s", e)
			return
		}

		refreshToken, err := exchangeCode(r.Context(), clientID, clientSecret, q.Get("code"), *redirectURI)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			result <- err
			return
		}

		fmt.Fprintln(w, "OK! ターミナルに表示されたリフレッシュトークンを使ってください。このタブは閉じて大丈夫です。")
		fmt.Printf("\nSPOTIFY_REFRESH_TOKEN=%s\n\n", refreshToken)
		fmt.Printf("authorized_at: %s（6ヶ月後までに再実行が必要）\n", time.Now().Format("2006-01-02"))
		result <- nil
	})

	srv := &http.Server{Handler: mux}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			result <- err
		}
	}()

	fmt.Println("ブラウザで以下の URL を開き、Spotify にログインして許可してください:")
	fmt.Println()
	fmt.Println(authURL)

	err = <-result
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

func exchangeCode(ctx context.Context, clientID, clientSecret, code, redirectURI string) (string, error) {
	form := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {redirectURI},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("token exchange: %s: %s", resp.Status, body)
	}

	var tok struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", fmt.Errorf("token exchange: decode: %w", err)
	}
	if tok.RefreshToken == "" {
		return "", errors.New("token exchange: response has no refresh_token")
	}
	return tok.RefreshToken, nil
}

func randomState() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(b)
}
