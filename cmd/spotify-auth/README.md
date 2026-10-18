# spotify-auth

Spotify セクション（よく聴くアーティスト・曲）で使うリフレッシュトークンを発行するコマンドです。

## トークンの更新

**リフレッシュトークンは認可から6ヶ月で失効します。**
失効するとサーバーログに `invalid_grant` が出て、セクションが非表示になるので、その前に差し替えてください。

### 1. トークンを発行する

```bash
SPOTIFY_CLIENT_ID=$(gcloud secrets versions access latest --secret=spotify-client-id --project portfolio-477702) \
SPOTIFY_CLIENT_SECRET=$(gcloud secrets versions access latest --secret=spotify-client-secret --project portfolio-477702) \
go run ./cmd/spotify-auth
```

表示された URL を開いて許可すると、ターミナルに `SPOTIFY_REFRESH_TOKEN=...` が出力されます。

### 2. Secret Manager に新しいバージョンを追加する

```bash
printf '%s' 'xxx' | gcloud secrets versions add spotify-refresh-token --data-file=- --project portfolio-477702
```

### 3. Cloud Run に反映する

新リビジョンを作らないと新しいトークンは読まれません。

```bash
gcloud run services update portfolio --region asia-northeast1 --project portfolio-477702 \
  --update-secrets=SPOTIFY_CLIENT_ID=spotify-client-id:latest,SPOTIFY_CLIENT_SECRET=spotify-client-secret:latest,SPOTIFY_REFRESH_TOKEN=spotify-refresh-token:latest
```

ローカルで確認する場合は `.env` の `SPOTIFY_REFRESH_TOKEN` も書き換えて `docker compose up --build` します。
