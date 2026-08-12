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
| `CORS_ALLOWED_ORIGINS` | 本番 ✅ | — | 許可 Origin（カンマ区切り） |
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
| `CORS_ALLOWED_ORIGINS` | `https://note-hub-three.vercel.app` | フロントからの API 通信 |
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

**CSRF / セッション:** Echo v4.15 の Fetch Metadata 方式を採用しています。`GET /api/auth/csrf` で CSRF トークンを事前取得し、`register` / `login` / `logout` では **Sec-Fetch-Site 検証 + Double Submit Cookie** を併用します。`refresh` は **Sec-Fetch-Site 検証 + Refresh Cookie** です。CSRF トークンは localStorage に保存せず、メモリ上のみ保持します。

動作確認: `https://<your-api>/health` → `{"status":"ok"}`

## CI

`main` / `develop` への push と PR で GitHub Actions が実行されます（Go vet・テスト、Frontend ビルド・テスト）。

## 関連ドキュメント

- [実装仕様書](doc/仕様書.md)
- [OpenAPI](backend/docs/openapi.yaml)

## ライセンス

未設定（必要に応じて追加してください）
