# Portfolio
GCPでデプロイしてます  
https://portfolio-15874071925.asia-northeast1.run.app/

以下のドメインでもアクセスできます  
https://umiyuri777.tech/

[この記事](https://qiita.com/umiyuri777/items/f5879348bea9e879fd3f)のサンプルです

## Spotify の再生傾向を表示する

`GET /me/top/{type}`（Spotify Web API）で、直近1ヶ月 / 半年 / 1年の
よく聴くアーティスト・曲の Top 10 を表示しています。
API は再生回数を返さないので、出しているのは順位だけです。
環境変数が未設定のときは、このセクションは表示されません。

### 1. Spotify アプリを作る

1. [Spotify Developer Dashboard](https://developer.spotify.com/dashboard) でアプリを作成（Web API を選択）
2. Redirect URI に `http://127.0.0.1:8888/callback` を登録（`localhost` は登録できません）
3. Client ID / Client Secret を控える

Development Mode のアプリはオーナーが Spotify Premium である必要があります。

### 2. リフレッシュトークンを発行する

```bash
SPOTIFY_CLIENT_ID=xxx SPOTIFY_CLIENT_SECRET=xxx go run ./cmd/spotify-auth
```

表示された URL を開いて許可すると、ターミナルに `SPOTIFY_REFRESH_TOKEN=...` が出力されます。
**リフレッシュトークンは認可から6ヶ月で失効する**ので、その前に再実行して差し替えてください
（失効するとサーバーログに `invalid_grant` が出て、セクションが非表示になります）。

### 3. ローカルで確認する

`.env`（git 管理外）に3つの値を書いて `docker compose up --build`。

```
SPOTIFY_CLIENT_ID=xxx
SPOTIFY_CLIENT_SECRET=xxx
SPOTIFY_REFRESH_TOKEN=xxx
```

### 4. Cloud Run に設定する

Secret Manager に登録し、サービスにマウントします（初回のみ。以降の `gcloud run deploy` でも設定は引き継がれます）。
Cloud Run の実行サービスアカウントには `roles/secretmanager.secretAccessor` が必要です。

```bash
printf '%s' 'xxx' | gcloud secrets create spotify-client-id --data-file=- --project portfolio-477702
printf '%s' 'xxx' | gcloud secrets create spotify-client-secret --data-file=- --project portfolio-477702
printf '%s' 'xxx' | gcloud secrets create spotify-refresh-token --data-file=- --project portfolio-477702

gcloud run services update portfolio --region asia-northeast1 --project portfolio-477702 \
  --update-secrets=SPOTIFY_CLIENT_ID=spotify-client-id:latest,SPOTIFY_CLIENT_SECRET=spotify-client-secret:latest,SPOTIFY_REFRESH_TOKEN=spotify-refresh-token:latest
```

トークンを更新したときは、新しいバージョンを追加してから同じ `gcloud run services update` を再実行（新リビジョン作成）します。

```bash
printf '%s' 'xxx' | gcloud secrets versions add spotify-refresh-token --data-file=- --project portfolio-477702
```
