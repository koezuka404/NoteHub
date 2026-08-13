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
| `CORS_ALLOWED_ORIGINS` | 本番 ✅ | — | 許可 Origin（カンマ区切り、本番 URL など） |
| `CORS_ALLOWED_ORIGIN_SUFFIXES` | — | — | 許可 Origin サフィックス（例: `.vercel.app` で Preview デプロイも許可） |
| `PUBLIC_HTTP_URL` | 推奨 | — | 公開 API URL（末尾スラッシュなし） |
| `REDIS_OPERATION_TIMEOUT` | — | `2s` | Redis 操作タイムアウト |
| `LOGIN_MAX_FAILURES` | — | `5` | ログイン失敗上限 |
| `LOGIN_FAILURE_WINDOW` | — | `1h` | 失敗カウント窓 |
| `LOGIN_LOCK_DURATION` | — | `1h` | アカウントロック時間 |
| `RATE_LIMIT_CAPACITY` | — | `10` | レートリミット容量 |
| `RATE_LIMIT_REFILL_PER_SECOND` | — | `1` | レートリミット補充率 |
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
| `VITE_WS_BASE_URL` | 本番 ✅ | `ws(s)://<host>` | WebSocket URL |
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
| `CORS_ALLOWED_ORIGINS` | `https://note-hub-three.vercel.app` | 本番フロント URL |
| `CORS_ALLOWED_ORIGIN_SUFFIXES` | `.vercel.app` | Vercel Preview デプロイ用（`note-hub-git-main-....vercel.app` 等） |
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
| **CORS** | ブラウザがクロスオリジン通信してよいか | `CORS_ALLOWED_ORIGINS`, `CORS_ALLOWED_ORIGIN_SUFFIXES` |
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
