package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type App struct {
	db       *Database
	store    *sessions.CookieStore
	templates *template.Template
}

type PageData struct {
	Title             string
	IsAuthenticated   bool
	CurrentUser       *User
	CurrentUserID     int
	Posts             []Post
	User              *User
	PostCount         int
	FollowerCount     int
	FollowingCount    int
	IsOwnProfile      bool
	IsFollowing       bool
	SuggestedUsers    []User
	Error             string
}

func main() {
	app := &App{
		store: sessions.NewCookieStore([]byte("your-session-secret-change-this")),
	}

	// データベース初期化
	db, err := sql.Open("sqlite3", "./gosns.db")
	if err != nil {
		log.Fatal("データベース接続エラー:", err)
	}
	defer db.Close()

	app.db = &Database{db}
	if err := app.db.CreateTables(); err != nil {
		log.Fatal("テーブル作成エラー:", err)
	}

	// テンプレート読み込み
	app.templates = template.Must(template.ParseGlob("templates/*.html"))

	// ルーター設定
	r := mux.NewRouter()

	// 静的ファイル
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads/"))))

	// 認証不要ページ
	r.HandleFunc("/", app.homeHandler).Methods("GET")
	r.HandleFunc("/login", app.loginHandler).Methods("GET", "POST")
	r.HandleFunc("/register", app.registerHandler).Methods("GET", "POST")
	r.HandleFunc("/auth/google", app.googleOAuthHandler).Methods("GET")
	r.HandleFunc("/auth/google/callback", app.googleCallbackHandler).Methods("GET")

	// 認証必要ページ
	r.HandleFunc("/logout", authMiddleware(app.logoutHandler)).Methods("GET")
	r.HandleFunc("/profile", authMiddleware(app.profileHandler)).Methods("GET")
	r.HandleFunc("/profile/{username}", authMiddleware(app.userProfileHandler)).Methods("GET")
	r.HandleFunc("/profile/update", authMiddleware(app.updateProfileHandler)).Methods("POST")
	r.HandleFunc("/posts", authMiddleware(app.createPostHandler)).Methods("POST")

	// API エンドポイント
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/posts", authMiddleware(app.getPostsAPI)).Methods("GET")
	api.HandleFunc("/posts/{id}/like", authMiddleware(app.likePostAPI)).Methods("POST")
	api.HandleFunc("/posts/{id}/comments", authMiddleware(app.getCommentsAPI)).Methods("GET")
	api.HandleFunc("/posts/{id}/comments", authMiddleware(app.createCommentAPI)).Methods("POST")
	api.HandleFunc("/posts/{id}", authMiddleware(app.deletePostAPI)).Methods("DELETE")
	api.HandleFunc("/users/{id}/follow", authMiddleware(app.followUserAPI)).Methods("POST")

	// サーバー起動
	fmt.Println("サーバーを起動中... http://podd.win:9090")
	log.Fatal(http.ListenAndServe(":9090", r))
}

func (app *App) homeHandler(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title: "ホーム",
	}

	// 認証状態確認
	userID := app.getCurrentUserID(r)
	if userID > 0 {
		data.IsAuthenticated = true
		data.CurrentUserID = userID
		
		// 現在のユーザー情報取得
		var user User
		err := app.db.QueryRow("SELECT id, username, email, avatar, bio FROM users WHERE id = ?", userID).
			Scan(&user.ID, &user.Username, &user.Email, &user.Avatar, &user.Bio)
		if err == nil {
			data.CurrentUser = &user
		}

		// タイムライン取得（フォローしているユーザーの投稿）
		data.Posts = app.getTimelinePosts(userID)

		// おすすめユーザー取得
		data.SuggestedUsers = app.getSuggestedUsers(userID, 5)
	} else {
		// 未認証の場合は全体の最新投稿を表示
		data.Posts = app.getLatestPosts(20)
	}

	app.renderTemplate(w, "home", data)
}

func (app *App) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data := PageData{Title: "ログイン"}
		app.renderTemplate(w, "login", data)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	var user User
	err := app.db.QueryRow("SELECT id, username, password FROM users WHERE email = ?", email).
		Scan(&user.ID, &user.Username, &user.Password)
	
	if err != nil || !checkPasswordHash(password, user.Password) {
		data := PageData{
			Title: "ログイン",
			Error: "メールアドレスまたはパスワードが正しくありません",
		}
		app.renderTemplate(w, "login", data)
		return
	}

	// JWT生成
	token, err := generateJWT(user.ID, user.Username)
	if err != nil {
		http.Error(w, "ログインに失敗しました", http.StatusInternalServerError)
		return
	}

	// Cookieに保存
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   false,
		MaxAge:   86400,
		Path:     "/",
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data := PageData{Title: "新規登録"}
		app.renderTemplate(w, "register", data)
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if password != confirmPassword {
		data := PageData{
			Title: "新規登録",
			Error: "パスワードが一致しません",
		}
		app.renderTemplate(w, "register", data)
		return
	}

	// パスワードハッシュ化
	hashedPassword, err := hashPassword(password)
	if err != nil {
		http.Error(w, "登録に失敗しました", http.StatusInternalServerError)
		return
	}

	// ユーザー作成
	_, err = app.db.Exec("INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
		username, email, hashedPassword)
	if err != nil {
		data := PageData{
			Title: "新規登録",
			Error: "そのユーザー名またはメールアドレスは既に使用されています",
		}
		app.renderTemplate(w, "register", data)
		return
	}

	// 作成したユーザーでログイン
	var userID int
	err = app.db.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&userID)
	if err != nil {
		http.Error(w, "ログインに失敗しました", http.StatusInternalServerError)
		return
	}

	// JWT生成
	token, err := generateJWT(userID, username)
	if err != nil {
		http.Error(w, "ログインに失敗しました", http.StatusInternalServerError)
		return
	}

	// Cookieに保存
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   false,
		MaxAge:   86400,
		Path:     "/",
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		HttpOnly: true,
		Secure:   false,
		MaxAge:   -1,
		Path:     "/",
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) profileHandler(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value("username").(string)
	
	http.Redirect(w, r, "/profile/"+username, http.StatusSeeOther)
}

func (app *App) userProfileHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]
	currentUserID := r.Context().Value("user_id").(int)

	// ユーザー情報取得
	var user User
	err := app.db.QueryRow("SELECT id, username, email, avatar, bio, created_at FROM users WHERE username = ?", username).
		Scan(&user.ID, &user.Username, &user.Email, &user.Avatar, &user.Bio, &user.CreatedAt)
	
	if err != nil {
		http.Error(w, "ユーザーが見つかりません", http.StatusNotFound)
		return
	}

	// 統計情報取得
	var postCount, followerCount, followingCount int
	app.db.QueryRow("SELECT COUNT(*) FROM posts WHERE user_id = ?", user.ID).Scan(&postCount)
	app.db.QueryRow("SELECT COUNT(*) FROM follows WHERE following_id = ?", user.ID).Scan(&followerCount)
	app.db.QueryRow("SELECT COUNT(*) FROM follows WHERE follower_id = ?", user.ID).Scan(&followingCount)

	// フォロー状態確認
	var isFollowing bool
	if currentUserID != user.ID {
		var count int
		app.db.QueryRow("SELECT COUNT(*) FROM follows WHERE follower_id = ? AND following_id = ?", 
			currentUserID, user.ID).Scan(&count)
		isFollowing = count > 0
	}

	// ユーザーの投稿取得
	posts := app.getUserPosts(user.ID, 20)

	data := PageData{
		Title:          user.Username + "のプロフィール",
		IsAuthenticated: true,
		CurrentUserID:  currentUserID,
		User:           &user,
		Posts:          posts,
		PostCount:      postCount,
		FollowerCount:  followerCount,
		FollowingCount: followingCount,
		IsOwnProfile:   currentUserID == user.ID,
		IsFollowing:    isFollowing,
	}

	app.renderTemplate(w, "profile", data)
}

func (app *App) updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	bio := r.FormValue("bio")

	// ファイルアップロード処理
	file, header, err := r.FormFile("avatar")
	avatarURL := ""
	if err == nil {
		defer file.Close()
		
		// ファイル保存
		filename := fmt.Sprintf("%d_%s", userID, header.Filename)
		filePath := filepath.Join("uploads", filename)
		
		dst, err := os.Create(filePath)
		if err == nil {
			defer dst.Close()
			io.Copy(dst, file)
			avatarURL = "/uploads/" + filename
		}
	}

	// プロフィール更新
	if avatarURL != "" {
		app.db.Exec("UPDATE users SET bio = ?, avatar = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", 
			bio, avatarURL, userID)
	} else {
		app.db.Exec("UPDATE users SET bio = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", 
			bio, userID)
	}

	username := r.Context().Value("username").(string)
	http.Redirect(w, r, "/profile/"+username, http.StatusSeeOther)
}

func (app *App) createPostHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	content := r.FormValue("content")

	// 画像アップロード処理
	file, header, err := r.FormFile("image")
	imageURL := ""
	if err == nil {
		defer file.Close()
		
		// ファイル保存
		filename := fmt.Sprintf("%d_%s_%s", userID, uuid.New().String(), header.Filename)
		filePath := filepath.Join("uploads", filename)
		
		dst, err := os.Create(filePath)
		if err == nil {
			defer dst.Close()
			io.Copy(dst, file)
			imageURL = "/uploads/" + filename
		}
	}

	// 投稿作成
	_, err = app.db.Exec("INSERT INTO posts (user_id, content, image_url) VALUES (?, ?, ?)",
		userID, content, imageURL)
	if err != nil {
		http.Error(w, "投稿に失敗しました", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) getCurrentUserID(r *http.Request) int {
	tokenString := ""
	if cookie, err := r.Cookie("token"); err == nil {
		tokenString = cookie.Value
	}

	if tokenString == "" {
		return 0
	}

	claims, err := validateJWT(tokenString)
	if err != nil {
		return 0
	}

	return claims.UserID
}

func (app *App) renderTemplate(w http.ResponseWriter, name string, data PageData) {
	err := app.templates.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}