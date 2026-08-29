# NoteHub

チーム向けのリアルタイム共同編集オンラインエディタです。ワークスペース単位でメンバーとドキュメントを管理し、WebSocket による同期・編集履歴・バージョン復元に対応しています。

## 主な機能

- ユーザー登録 / ログイン（JWT + Refresh Token + CSRF）
- ワークスペースの作成・管理
- メンバーの招待・停止・復帰（オーナー権限）
- ドキュメントの作成・編集・削除
- Monaco Editor によるリアルタイム共同編集
- 手動保存・自動保存・編集履歴・復元

## 技術スタック

| 層 | 技術 |
|---|---|
| Frontend | React, TypeScript, Vite, React Router, Monaco Editor, Tailwind CSS |
| Backend | Go, Echo, GORM |
| DB | PostgreSQL |
| Cache / Session | Redis |
| リアルタイム | WebSocket |

## リポジトリ構成

```
NoteHub/
├── backend/          # Go API + WebSocket
├── frontend/         # React SPA
├── doc/              # 仕様書
├── docker-compose.yml
└── render.yaml       # Render Blueprint（本番バックエンド）
```

## ローカル開発

### 前提

- Docker / Docker Compose

### 環境変数のセットアップ

命名は **Dotenv / Vite の慣習** に従います。

| ファイル | コミット | 役割 |
|---|---|---|
| `.env.example` | ✅ | **定義例** — 変数名・デフォルト・説明（テンプレート） |
| `.env.local` | ❌ | **実際の値** — シークレットとローカル上書き |

```bash
cp .env.example .env.local
# .env.local の JWT_SECRET を 32 文字以上に変更
# 例: openssl rand -base64 48
docker compose up -d
```

モノレポ内の定義例:

| パス | 対象 |
|---|---|
| [`.env.example`](.env.example) | Docker ローカル開発（Backend 全パラメータ） |
| [`backend/.env.example`](backend/.env.example) | Render デプロイ |
| [`frontend/.env.example`](frontend/.env.example) | Vercel / `npm run dev` |

ローカル Docker では `docker-compose.yml` が非シークレットのデフォルトを設定し、`.env.local` から `JWT_SECRET` などを読み込みます。

### 起動

```bash
docker compose up -d
```

| サービス | URL |
|---|---|
| Frontend | http://localhost:5173 |
| Backend API | http://localhost:8081 |
| Health check | http://localhost:8081/health |

初回起動時、フロントエンドコンテナ内で `npm install` が走るため、表示まで 1〜2 分かかることがあります。

### 停止

```bash
docker compose down
```

## テスト

### Backend

```bash
cd backend
go test ./...
```

### Frontend

```bash
cd frontend
npm install
npm test
npm run test:coverage
```

## 環境変数リファレンス

迅速な検証・デプロイのため、主要パラメータを一覧にまとめています。詳細は各 `.env.example` も参照してください。

### Backend（`backend/config/env.go`）

| 変数 | 必須 | デフォルト（未設定時） | 説明 |
|---|---|---|---|
| `JWT_SECRET` | ✅ | — | JWT 署名鍵（**32 文字以上**。`.env.local` または Render Dashboard） |
| `APP_ENV` | — | `development` | `development` / `test` / `production` |
| `DATABASE_URL` | 本番 ✅ | — | PostgreSQL 接続 URL |
| `REDIS_URL` | 本番 ✅ | — | Redis 接続 URL |
| `HTTP_PORT` | — | `8080` | ローカル API ポート（Render は `PORT` を使用） |
| `JWT_ISSUER` | — | `notehub-api` | JWT `iss` |
| `JWT_AUDIENCE` | — | `notehub-client` | JWT `aud` |
| `ACCESS_TOKEN_TTL_MINUTES` | — | `15` | アクセストークン TTL（分） |
| `REFRESH_TOKEN_TTL_MINUTES` | — | `30` | 開発環境リフレッシュ TTL（分） |
| `REFRESH_TOKEN_TTL_DAYS` | — | `14` | 本番リフレッシュ TTL（日） |
| `BCRYPT_COST` | — | `12` | bcrypt コスト |
| `COOKIE_SECURE` | — | 本番 `true` / 開発 `false` | HTTPS Cookie |
| `COOKIE_SAME_SITE` | — | `Lax` | 本番クロスオリジン時は `None` |
| `COOKIE_DOMAIN` | — | 空 | Cookie ドメイン |
| `REFRESH_TOKEN_COOKIE_NAME` | — | `notehub_refresh_token` | Refresh Cookie 名 |
| `CSRF_TOKEN_COOKIE_NAME` | — | `notehub_csrf_token` | CSRF Cookie 名 |
| `MAX_REQUEST_BODY_BYTES` | — | `2097152`（2MiB） | リクエストボディ上限。超過は 413。巨大ボディによる課金・DoS 対策 |
| `CORS_ALLOWED_ORIGINS` | 本番 ✅ | — | 許可 Origin（カンマ区切り、完全一致のみ。サブドメインは別途列挙） |
| `PUBLIC_HTTP_URL` | 推奨 | — | 公開 API URL（末尾スラッシュなし） |
| `REDIS_OPERATION_TIMEOUT` | — | `2s` | Redis 操作タイムアウト |
| `LOGIN_MAX_FAILURES` | — | `5` | ログイン失敗上限 |
| `LOGIN_FAILURE_WINDOW` | — | `1h` | 失敗カウント窓 |
| `LOGIN_LOCK_DURATION` | — | `1h` | アカウントロック時間 |
| `RATE_LIMIT_CAPACITY` | — | `10` | レートリミット容量 |
| `RATE_LIMIT_REFILL_PER_SECOND` | — | `1` | レートリミット補充率 |
| `TRUSTED_PROXY_CIDRS` | — | 空 | クライアント IP ヘッダを信じてよいプロキシ CIDR（カンマ区切り）。空なら `X-Forwarded-For` 等は無視し TCP ピアを使う |
| `CLIENT_IP_HEADER` | — | `X-Vercel-Forwarded-For` | 信頼できるプロキシからのみ読むクライアント IP ヘッダ |
| `WS_MAX_CONNECTIONS_PER_DOCUMENT` | — | `3` | ドキュメントあたり WS 接続上限 |
| `DOCUMENT_AUTOSAVE_INTERVAL` | — | `10s` | 自動保存間隔 |
| `DOCUMENT_AUTOSAVE_IDLE_DURATION` | — | `60s` | アイドル自動保存 |
| `CLEANUP_BATCH_INTERVAL` | — | `1h` | クリーンアップバッチ間隔 |
| `REFRESH_TOKEN_RETENTION` | — | `720h` | Refresh トークン保持期間 |
| `BACKUP_ENABLED` | — | `false` | DB バックアップ有効化 |
| `BACKUP_BATCH_INTERVAL` | — | `24h` | バックアップ間隔 |
| `BACKUP_DIR` | — | `./backups` | バックアップ出力先 |
| `BACKUP_RETENTION` | — | `168h` | バックアップ保持期間 |

### Frontend（Vite）

| 変数 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `VITE_API_BASE_URL` | 本番 ✅ | 空（同一オリジン `/api`） | Render API URL（末尾スラッシュなし） |
| `VITE_WS_BASE_URL` | 本番 ✅ | `ws(s)://<host>` | WebSocket のホスト（トークンは URL に含めない。サブプロトコルで渡す） |
| `VITE_PROXY_TARGET` | — | `http://localhost:8080` | Docker 内 Vite プロキシ先（`vite.config.ts`） |

### ローカル Docker クイック検証

```bash
cp .env.example .env.local   # JWT_SECRET を編集
docker compose up -d
curl -s http://localhost:8081/health   # {"status":"ok"}
open http://localhost:5173
```

| 確認項目 | URL / コマンド | 期待結果 |
|---|---|---|
| API ヘルス | `curl http://localhost:8081/health` | `{"status":"ok"}` |
| フロント | http://localhost:5173 | ログイン画面 |
| Postgres | `localhost:5436` | docker-compose の DB |
| Redis | `localhost:6382` | docker-compose の Redis |

## 本番デプロイ

**Frontend → Vercel** / **Backend → Render** の構成を想定しています。

### 1. Render（Backend）

1. [Render Dashboard](https://dashboard.render.com) → **New → Blueprint**
2. このリポジトリを接続し `render.yaml` を適用
3. 作成されるリソース: `notehub-api`, `notehub-db`, `notehub-redis`
4. `notehub-api` の Environment に設定:

| 変数 | 必須 | 説明 |
|---|---|---|
| `JWT_SECRET` | ✅ | Render が自動生成可（`render.yaml`） |
| `PUBLIC_HTTP_URL` | ✅ | Render の公開 URL（例: `https://notehub-api.onrender.com`） |
| `CORS_ALLOWED_ORIGINS` | ✅ | Vercel のフロント URL（HTTPS） |
| `COOKIE_SECURE` | — | `true`（Blueprint 既定） |
| `COOKIE_SAME_SITE` | — | `None`（Blueprint 既定） |

`DATABASE_URL` / `REDIS_URL` / `APP_ENV` は Blueprint で自動設定されます。

詳細は [`.env.example`](.env.example) と [`backend/.env.example`](backend/.env.example) を参照してください。

### 2. Vercel（Frontend）

1. [Vercel](https://vercel.com) でリポジトリをインポート
2. **Root Directory** を `frontend` に設定
3. Environment Variables:

| 変数 | 必須 | 説明 |
|---|---|---|
| `VITE_API_BASE_URL` | ✅ | Render API URL（末尾スラッシュなし） |
| `VITE_WS_BASE_URL` | ✅ | WebSocket URL（`wss://`） |

詳細は [`frontend/.env.example`](frontend/.env.example) を参照してください。

4. デプロイ:

```bash
cd frontend
vercel login
vercel deploy --prod
```

### 3. 仕上げ

Vercel の本番 URL が確定したら、Render の Environment を設定して API を再デプロイしてください。

| 変数 | 例 | 用途 |
|------|-----|------|
| `CORS_ALLOWED_ORIGINS` | `https://note-hub-three.vercel.app` | 本番フロント URL（完全一致。Preview 用ホストはカンマで追加） |
| `TRUSTED_PROXY_CIDRS` | Vercel Static IPs の CIDR | クライアント IP ヘッダを信じる送信元（空ならヘッダ無視） |
| `CLIENT_IP_HEADER` | `X-Vercel-Forwarded-For` | Vercel が付与するクライアント IP 専用ヘッダ |
| `PUBLIC_HTTP_URL` | `https://notehub-4uet.onrender.com` | **Render Dashboard に表示される実際の URL** |
| `APP_ENV` | `production` | 本番設定 |
| `COOKIE_SECURE` | `true` | HTTPS Cookie |
| `COOKIE_SAME_SITE` | `None` | クロスオリジン Cookie（CSRF / Refresh 必須） |
| `REDIS_URL` | Redis Internal URL | 本番必須 |

Vercel の Environment:

| 変数 | 例 |
|------|-----|
| `VITE_API_BASE_URL` | `https://notehub-4uet.onrender.com` |
| `VITE_WS_BASE_URL` | `wss://notehub-4uet.onrender.com` |

**注意:** `notehub-api.onrender.com` など Blueprint 名と異なる URL になることがあります。Dashboard の URL を使ってください。

**CSRF / セッション:** Echo v4.15 方式。`Sec-Fetch-Site` が `same-origin` / `none` なら Fetch Metadata で許可、`cross-site` / `same-site` では **Double Submit Cookie にフォールバック**します（Vercel + Render のクロスオリジン構成向け）。`register` / `login` / `logout` は CSRF 必須、`refresh` は Refresh Cookie で保護します。

### Sec-Fetch-Site 導入の軌跡

認証 API の CSRF 対策は、本番デプロイ（Vercel + Render）を通じて段階的に整理されました。

#### 1. Origin ミドルウェア（初期）

`refresh` / `logout` 向けに `Origin` ヘッダと `CORS_ALLOWED_ORIGINS` を照合する `origin.go` を導入しました。同一オリジン前提の CSRF 補助として機能していました。

#### 2. Origin + CSRF 統合（`origin_or_csrf.go`）

`register` / `login` 追加に伴い、Origin 検証と Double Submit CSRF を 1 つのミドルウェアにまとめた `origin_or_csrf.go` を追加しました。ルートごとに適用ミドルウェアが分かれ、責務が曖昧になったため、次の段階で分割しました。

#### 3. Echo v4.15 方式への移行（`Sec-Fetch-Site` + CSRF 分離）

[Echo v4.15](https://github.com/labstack/echo/releases/tag/v4.15.0) の Fetch Metadata 方式を採用し、ミドルウェアを分離しました。

| ファイル | 役割 |
|---|---|
| `middleware/sec_fetch_site.go` | `Sec-Fetch-Site` ヘッダによる Fetch Metadata 検証 |
| `middleware/csrf.go` | Double Submit Cookie（`notehub_csrf_token` + `X-CSRF-Token`） |
| `middleware/cors.go` | ブラウザのクロスオリジン通信許可（別レイヤー） |

削除した旧ファイル: `origin.go`, `origin_or_csrf.go` および各テスト。

`GET /api/auth/csrf` を追加し、フロントは `LoginPage` 表示時に CSRF トークンを先読みします（`frontend/src/api.ts` の `ensureCsrfToken()`）。

#### 4. 本番 403（`SEC_FETCH_SITE_BLOCKED`）と修正

初版の `SecFetchSite` は `cross-site` / `same-site` を **403 で拒否**していました。Vercel（フロント）→ Render（API）はブラウザから常に `Sec-Fetch-Site: cross-site` になるため、本番で `GET /api/auth/csrf` や `POST /auth/login` が失敗しました。

```
ブラウザ (Vercel)  →  Sec-Fetch-Site: cross-site  →  旧 SecFetchSite: 403
                                                    →  CORS は別問題（環境変数で解決）
```

#### 5. 現行設計（Echo フォールバック）

Echo 公式と同様、**Fetch Metadata で確証できる場合のみ早期許可**し、それ以外は CSRF に委ねます。

| `Sec-Fetch-Site` | SecFetchSite ミドルウェア | 次のチェック |
|---|---|---|
| `same-origin` / `none` | 許可（`sec_fetch_site_validated` をセット） | CSRF ルートなら Double Submit |
| `cross-site` / `same-site` | **通過（403 しない）** | CSRF ルートなら **Double Submit 必須** |
| ヘッダなし | 通過 | curl 等は CSRF で保護 |
| 未知の値 | 403 `SEC_FETCH_SITE_BLOCKED` | — |

#### 配線（現行）

ミドルウェアの **生成** は `main.go`、**ルートへの適用** は `router/router.go` です。

```go
// main.go — ミドルウェア生成
secFetchSiteMiddleware := appmiddleware.NewSecFetchSiteMiddleware()
csrfMiddleware := appmiddleware.NewCSRFMiddleware(...)

// router/router.go — 認証ルート
api.GET("/auth/csrf",     deps.Auth.IssueCSRF, deps.SecFetchSite, deps.RateLimit)
api.POST("/auth/register", deps.Auth.Register, deps.SecFetchSite, deps.CSRF, deps.RateLimit)
api.POST("/auth/login",    deps.Auth.Login,    deps.SecFetchSite, deps.CSRF, deps.RateLimit)
api.POST("/auth/refresh",  deps.Auth.Refresh,  deps.RequireRefreshToken, deps.SecFetchSite, deps.RateLimit)
api.POST("/auth/logout",   deps.Auth.Logout,   deps.AuthMiddleware, deps.SecFetchSite, deps.CSRF, deps.RateLimit)
```

#### レイヤー整理

| レイヤー | 役割 | 設定 |
|---|---|---|
| **CORS** | ブラウザがクロスオリジン通信してよいか | `CORS_ALLOWED_ORIGINS`（完全一致） |
| **Body limit** | リクエストボディサイズ | `MAX_REQUEST_BODY_BYTES` |
| **Rate limit IP** | 制限キーのクライアント IP | `TRUSTED_PROXY_CIDRS` + `CLIENT_IP_HEADER` |
| **WebSocket 認証** | 接続時のアクセストークン | `Sec-WebSocket-Protocol: bearer, <JWT>` |
| **Sec-Fetch-Site** | Fetch Metadata による早期許可 | コード（環境変数不要） |
| **CSRF** | 状態変更リクエストの正当性 | Cookie + `X-CSRF-Token` |
| **Refresh Cookie** | セッション更新 | HttpOnly Cookie（`/api/auth/refresh`） |

本番で 403 が出る場合は Response body の `error.code` を確認してください（`SEC_FETCH_SITE_BLOCKED` / `CSRF_TOKEN_REQUIRED` / `CSRF_TOKEN_INVALID`）。

動作確認: `https://<your-api>/health` → `{"status":"ok"}`

## CI

`main` / `develop` への push と PR で GitHub Actions が実行されます（Go vet・テスト、Frontend ビルド・テスト）。

## 関連ドキュメント

- [実装仕様書](doc/仕様書.md)
- [OpenAPI](backend/docs/openapi.yaml)

## ライセンス

未設定（必要に応じて追加してください）

## セキュリティまわりの修正

レート制限のクライアント IP、CORS、リクエストボディ上限、WebSocket 認証、パスワード文字数判定を整理した。変更したファイルは各項に列挙する。

### レート制限のクライアント IP

問題は、`X-Forwarded-For` を誰でも付けられること。無条件に信じると他人の IP を偽ってレート制限を回避できる。Echo 標準の `ExtractIPFromXFFHeader()` は使わない。

**判定**

| 条件 | 使う IP |
|---|---|
| 接続元が `TRUSTED_PROXY_CIDRS` に含まれる | `CLIENT_IP_HEADER`（既定 `X-Vercel-Forwarded-For`） |
| それ以外、またはヘッダーが無い／不正 | TCP のピアアドレス（`RemoteAddr`） |

`X-Forwarded-For` は、信頼できるプロキシから来ていても使わない。Vercel からのクライアント IP だけを専用ヘッダーで読む。

**修正ファイル**

| ファイル | 内容 |
|---|---|
| `backend/router/router.go` | `e.IPExtractor` に `NewClientIPExtractor(cfg.TrustedProxyCIDRs, cfg.ClientIPHeader)` を渡す。ヘッダー名は直書きしない。レート制限の `RealIP()` もこの結果を使う |
| `backend/middleware/client_ip.go` | 信頼 CIDR が空 → ヘッダーは見ない。ピアが CIDR 外 → `RemoteAddr`。ヘッダー名が空または `X-Forwarded-For` → `X-Vercel-Forwarded-For` に置き換え |
| `backend/config/env.go` | `CLIENT_IP_HEADER` を読む（未設定なら `X-Vercel-Forwarded-For`）。値に `X-Forwarded-For` を指定すると起動時エラー |
| `backend/middleware/client_ip_test.go` | なりすましヘッダー拒否・専用ヘッダー採用のテスト |
| `backend/router/router_test.go` | 設定した専用ヘッダーを使うこと、`X-Forwarded-For` を無視すること |

本番では `TRUSTED_PROXY_CIDRS` が必須。Vercel Static IP の CIDR を入れたときだけ、専用ヘッダーのクライアント IP を信じる。ブラウザが Render API へ直接（CORS）接続する現行構成では、リクエストは Vercel を経由しないので、未設定のまま（ヘッダ無視）が正しい。

### CORS（サブドメイン一括許可の廃止）

以前は次の 2 段だった。

1. `AllowedOrigins` に書いてある Origin と一致したら許可
2. 一致しなくても `https://` で、かつ Origin が `.vercel.app` のようなサフィックスで終われば許可

2 があると、本番フロントが `https://note-hub-three.vercel.app` でも `https://なんでも.vercel.app` が通る。Preview 用ホストや別プロジェクトの Vercel アプリまで CORS 対象になる。

今はマップの **完全一致だけ**。

```go
AllowOriginFunc: func(origin string) (bool, error) {
    _, ok := allowed[origin]
    return ok, nil
}
```

開発時に Origin 未設定なら、これまでどおり `http://localhost:5173` と `http://127.0.0.1:5173` だけ。メソッド・ヘッダー・Cookie の扱いは変えていない。

**修正ファイル**

| ファイル | 内容 |
|---|---|
| `backend/middleware/cors.go` | サフィックス照合を削除。完全一致のみ |
| `backend/config/env.go` | `AllowedOriginSuffixes` と `CORS_ALLOWED_ORIGIN_SUFFIXES` の読み込み・本番バリデーションを削除 |
| `backend/middleware/cors_test.go` | サフィックス許可テストをやめ、許可リストに無いサブドメイン（例: `https://preview.note-hub-three.vercel.app`）は `Access-Control-Allow-Origin` を付けないことを確認 |
| `backend/.env.example` | `CORS_ALLOWED_ORIGIN_SUFFIXES=.vercel.app` を削除 |
| `render.yaml` | 同じ環境変数を削除 |

Preview デプロイを許可したい場合は、その URL を `CORS_ALLOWED_ORIGINS` に明示的に足す。`.vercel.app` 全体は開かない。

### リクエストボディ上限（DoS / 転送課金対策）

上限なし（またはハンドラで初めて読む）だと、攻撃者が何 GB でも送れる。クラウドでは受信バイトに課金されることがあり、DoS がそのまま請求になる。ボディをラップするだけでは、`Bind` 失敗を各 API が 400 にしてしまい 413 にならないことがあった。

グローバルミドルウェアでコントローラの前に切る。上限は `MAX_REQUEST_BODY_BYTES`（未設定なら **2MiB**、設定できる範囲は 1KiB〜64MiB）。

```go
// backend/router/router.go — ログの直後・CORS の前
e.Use(appmiddleware.NewBodyLimitMiddleware(cfg.MaxRequestBodyBytes))
```

`backend/middleware/body_limit.go` の流れ:

1. GET / HEAD / OPTIONS などは対象外（WebSocket の upgrade は GET なので検査しない）
2. `Content-Length` が上限超え → 本体を読まずすぐ **413** `REQUEST_BODY_TOO_LARGE`
3. 長さが無い／申告が小さい（chunked など） → `http.MaxBytesReader` で最大バイトまでだけ読む。超えたらハンドラを呼ばず 413。`Response` を渡しているので超過時に接続側も止めやすい
4. 上限以内 → 読んだバイト列を新しい `Body` に載せ替えてから次のハンドラへ。`Bind` は小さいバッファだけ見る

超過時のメッセージは「リクエストが大きすぎます」。ドキュメント本文のアプリ側制限とは別の、HTTP 層の受け取り上限。

**修正ファイル**

| ファイル | 内容 |
|---|---|
| `backend/router/router.go` | `NewBodyLimitMiddleware` を全ルートに適用 |
| `backend/middleware/body_limit.go` | サイズ検査・`MaxBytesReader`・413 応答 |
| `backend/middleware/body_limit_test.go` | Content-Length 超過、実読込超過、GET スキップ |
| `backend/router/router_test.go` | 過大 POST で 413 |
| `backend/.env.example` | `MAX_REQUEST_BODY_BYTES=2097152` |

### WebSocket 認証（サブプロトコル）

ブラウザの `WebSocket` は handshake に任意ヘッダーを付けられない。トークンをクエリに載せるとログや Referer、プロキシ履歴に残る。認証用の文字列は `Sec-WebSocket-Protocol` に載せ、接続後はそのサブプロトコル `bearer` でデータを送る。

以前（フロント）:

```ts
new WebSocket(`${url}?access_token=${token}`)
```

現在（`frontend/src/documentWs.ts` / `workspaceWs.ts`）:

```ts
const url = `${wsBaseUrl()}/ws/documents/${documentId}`;
const ws = new WebSocket(url, ['bearer', token]);
```

ブラウザは次のヘッダーにする。

```
Sec-WebSocket-Protocol: bearer, <JWT>
```

サーバー（`backend/controller/websocket.go`）はサブプロトコルからだけトークンを取る。`bearer` はスキップし、`.` が 2 つある JWT らしき値だけをトークンにする。`?access_token=` と `Authorization` は見ない。

Upgrade 時にサブプロトコル `bearer` を選んで返す（JWT 自体はプロトコル名にしない）。

```go
wsUpgrader = func(checkOrigin func(*http.Request) bool) *gorillaws.Upgrader {
    return &gorillaws.Upgrader{
        CheckOrigin:  checkOrigin,
        Subprotocols: []string{websocketAuthSubprotocol},
    }
}
```

流れ:

1. クライアントが `bearer` + トークン文字列を送る
2. サーバーが JWT を検証してから Upgrade
3. 応答で `Sec-WebSocket-Protocol: bearer`
4. その上で JSON イベントを送受信

本番の `ws://` は拒否（`403 WEBSOCKET_TLS_REQUIRED`）。`wss://` 必須。

**修正ファイル**

| ファイル | 内容 |
|---|---|
| `backend/controller/websocket.go` | トークン抽出、`Subprotocols: ["bearer"]` |
| `backend/controller/websocket_test.go` | サブプロトコル JWT、選択プロトコルが `bearer` |
| `frontend/src/documentWs.ts` | `new WebSocket(url, ['bearer', token])` |
| `frontend/src/workspaceWs.ts` | 同上 |
| `frontend/src/test/mocks/websocket.ts` | `protocols` を保持 |
| `frontend/src/documentWs.test.ts` / `workspaceWs.test.ts` | URL にトークンが無いこと、`['bearer', token]` で接続すること |
| `backend/docs/openapi.yaml` | クエリ認証から `Sec-WebSocket-Protocol` に変更 |

### パスワード長（文字数判定）

Go の `len(string)` は UTF-8 の **バイト数** なので、日本語 8 文字でも制限を超えて落ちていた。標準の `strings` に長さ関数はないため、表示名と同じ `utf8.RuneCountInString` で **文字数（ルーン数）** を数える。8〜15 はこれまでどおり文字数。マルチバイトでも 8 文字なら通り、15 文字超は弾く。

**修正ファイル**

| ファイル | 内容 |
|---|---|
| `backend/usecase/auth.go` | `validatePassword` の長さ判定を `utf8.RuneCountInString` に変更 |
| `backend/usecase/auth_extended_test.go` | マルチバイト 8 文字は可、15 文字超は不可 |
