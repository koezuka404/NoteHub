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
- ルートに `.env`（`JWT_SECRET` など。32 文字以上必須）

`.env` の例:

```env
JWT_SECRET=notehub-development-secret-key-32bytes-minimum
APP_ENV=development
```

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

## 本番デプロイ

**Frontend → Vercel** / **Backend → Render** の構成を想定しています。

### 1. Render（Backend）

1. [Render Dashboard](https://dashboard.render.com) → **New → Blueprint**
2. このリポジトリを接続し `render.yaml` を適用
3. 作成されるリソース: `notehub-api`, `notehub-db`, `notehub-redis`
4. `notehub-api` の Environment に設定:

| 変数 | 説明 |
|---|---|
| `PUBLIC_HTTP_URL` | Render の公開 URL（例: `https://notehub-api.onrender.com`） |
| `CORS_ALLOWED_ORIGINS` | Vercel のフロント URL（HTTPS） |

詳細は [`backend/.env.example`](backend/.env.example) を参照してください。

### 2. Vercel（Frontend）

1. [Vercel](https://vercel.com) でリポジトリをインポート
2. **Root Directory** を `frontend` に設定
3. Environment Variables:

| 変数 | 説明 |
|---|---|
| `VITE_API_BASE_URL` | Render API URL（末尾スラッシュなし） |
| `VITE_WS_BASE_URL` | WebSocket URL（`wss://`） |

4. デプロイ:

```bash
cd frontend
vercel login
vercel deploy --prod
```

詳細は [`frontend/.env.example`](frontend/.env.example) を参照してください。

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

**CSRF（Double Submit Cookie）:** ログイン / リフレッシュ時に API が返す `csrfToken` をフロントが `sessionStorage` に保存し、`X-CSRF-Token` ヘッダーで送信します。Cookie が送れない場合（`COOKIE_SAME_SITE` 未設定など）は `/api/auth/refresh` が 403 になります。

動作確認: `https://<your-api>/health` → `{"status":"ok"}`

## CI

`main` / `develop` への push と PR で GitHub Actions が実行されます（Go vet・テスト、Frontend ビルド・テスト）。

## 関連ドキュメント

- [実装仕様書](doc/仕様書.md)
- [OpenAPI](backend/docs/openapi.yaml)

## ライセンス

未設定（必要に応じて追加してください）
