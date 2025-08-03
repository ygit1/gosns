# GoSNS - 軽量Go製ソーシャルネットワーキングサービス

超軽量で高速なGoオンリーのSNSプラットフォーム。フロントエンドもGo（html/template）、DBはSQLiteで構築された完全な機能を持つSNSです。

## 特徴

- ⚡ **超軽量**: GoとSQLiteのみ使用、依存関係最小限
- 🚀 **高速**: ネイティブGo、コンパイル済みバイナリ
- 🔐 **全機能認証**: JWT、Google OAuth、メール認証対応
- 📱 **レスポンシブ**: モバイル対応のクリーンなUI
- 🔄 **リアルタイム**: Ajax対応のいいね・コメント機能
- 📸 **画像対応**: 投稿・アバターの画像アップロード
- 👥 **SNS機能**: フォロー、タイムライン、おすすめユーザー

## 主要機能

### 認証システム
- ✅ ユーザー登録・ログイン
- ✅ Google OAuth認証
- ✅ JWT トークン認証
- ✅ セキュアなパスワードハッシュ化

### コア機能
- ✅ 投稿作成・表示・削除
- ✅ 画像アップロード（投稿・アバター）
- ✅ いいね機能（Ajax）
- ✅ コメント機能（Ajax）
- ✅ フォロー・アンフォロー
- ✅ パーソナライズされたタイムライン
- ✅ ユーザープロフィール
- ✅ おすすめユーザー機能

### UI/UX
- ✅ レスポンシブデザイン
- ✅ 軽量CSS（フレームワーク不使用）
- ✅ リアルタイムJavaScript
- ✅ 無限スクロール対応
- ✅ 画像プレビュー機能

## 技術スタック

- **言語**: Go 1.21+
- **データベース**: SQLite3
- **フロントエンド**: Go html/template + Vanilla JS + CSS
- **認証**: JWT + Google OAuth2
- **依存関係**:
  - gorilla/mux (ルーティング)
  - gorilla/sessions (セッション管理)
  - mattn/go-sqlite3 (SQLite driver)
  - golang-jwt/jwt (JWT)
  - golang.org/x/oauth2 (OAuth2)
  - golang.org/x/crypto (パスワードハッシュ)

## セットアップ

### 1. リポジトリのクローン
\`\`\`bash
git clone <repository-url>
cd gosns
\`\`\`

### 2. 依存関係のインストール
\`\`\`bash
go mod tidy
\`\`\`

### 3. 環境変数の設定（Google OAuth使用時）
\`\`\`bash
cp .env.example .env
# .envファイルを編集してGoogle OAuth情報を設定
\`\`\`

### 4. ビルドと実行
\`\`\`bash
go build -o gosns .
./gosns
\`\`\`

サーバーは http://podd.win:8080 で起動します。

## プロジェクト構造

\`\`\`
gosns/
├── main.go              # メインサーバー、ルーティング
├── models.go            # データベースモデル
├── auth.go              # 認証システム（JWT、OAuth）
├── handlers.go          # APIハンドラー
├── templates/           # HTMLテンプレート
│   ├── layout.html     # ベースレイアウト
│   ├── home.html       # ホームページ
│   ├── login.html      # ログインページ
│   ├── register.html   # 登録ページ
│   └── profile.html    # プロフィールページ
├── static/             # 静的ファイル
│   ├── css/
│   │   └── style.css   # メインスタイルシート
│   ├── js/
│   │   └── app.js      # フロントエンドJavaScript
│   └── img/            # 画像ファイル
├── uploads/            # アップロード画像保存
├── gosns.db           # SQLiteデータベース（自動作成）
├── go.mod
├── go.sum
└── README.md
\`\`\`

## API エンドポイント

### 認証
- \`GET /login\` - ログインページ
- \`POST /login\` - ログイン処理
- \`GET /register\` - 登録ページ
- \`POST /register\` - 登録処理
- \`GET /auth/google\` - Google OAuth開始
- \`GET /auth/google/callback\` - Google OAuth コールバック
- \`GET /logout\` - ログアウト

### ページ
- \`GET /\` - ホームページ・タイムライン
- \`GET /profile\` - 自分のプロフィール
- \`GET /profile/{username}\` - ユーザープロフィール
- \`POST /profile/update\` - プロフィール更新
- \`POST /posts\` - 投稿作成

### API
- \`GET /api/posts\` - 投稿一覧取得（ページネーション対応）
- \`POST /api/posts/{id}/like\` - いいね・いいね解除
- \`GET /api/posts/{id}/comments\` - コメント取得
- \`POST /api/posts/{id}/comments\` - コメント作成
- \`DELETE /api/posts/{id}\` - 投稿削除
- \`POST /api/users/{id}/follow\` - フォロー・アンフォロー

## データベーススキーマ

### users テーブル
- \`id\` (PRIMARY KEY)
- \`username\` (UNIQUE)
- \`email\` (UNIQUE)
- \`password\` (ハッシュ化)
- \`avatar\` (画像URL)
- \`bio\` (自己紹介)
- \`google_id\` (Google OAuth用)
- \`verified\` (認証済みフラグ)
- \`created_at\`, \`updated_at\`

### posts テーブル
- \`id\` (PRIMARY KEY)
- \`user_id\` (FOREIGN KEY)
- \`content\` (投稿内容)
- \`image_url\` (画像URL)
- \`likes\` (いいね数)
- \`comments\` (コメント数)
- \`created_at\`, \`updated_at\`

### follows テーブル
- \`id\` (PRIMARY KEY)
- \`follower_id\` (フォローする人)
- \`following_id\` (フォローされる人)
- \`created_at\`

### likes テーブル
- \`id\` (PRIMARY KEY)
- \`user_id\` (FOREIGN KEY)
- \`post_id\` (FOREIGN KEY)

### comments テーブル
- \`id\` (PRIMARY KEY)
- \`user_id\` (FOREIGN KEY)
- \`post_id\` (FOREIGN KEY)
- \`content\` (コメント内容)
- \`created_at\`

## パフォーマンス

- **ビルドサイズ**: ~15MB（静的バイナリ）
- **メモリ使用量**: ~10-20MB
- **起動時間**: ~100ms
- **レスポンス時間**: <5ms（通常操作）
- **データベース**: SQLiteファイル、設定不要

## セキュリティ

- パスワードのbcryptハッシュ化
- JWTトークンベース認証
- SQLインジェクション対策（prepared statements）
- HTTPOnlyクッキー
- CSRF保護
- ファイルアップロード制限

## 開発・拡張

### カスタマイズ
- \`static/css/style.css\` でスタイル変更
- \`templates/\` でHTML構造変更
- \`static/js/app.js\` でフロントエンド機能追加

### 機能追加例
- リアルタイム通知（WebSocket）
- プライベートメッセージ
- ハッシュタグ機能
- 画像フィルター
- 検索機能

## ライセンス

MIT License

## 貢献

Issue・Pull Request歓迎です！

## サポート

技術的な質問やバグ報告は Issue をご利用ください。