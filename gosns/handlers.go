package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// API レスポンス用構造体
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Posts   []Post      `json:"posts,omitempty"`
	Comments []Comment  `json:"comments,omitempty"`
	Likes   int         `json:"likes,omitempty"`
	Liked   bool        `json:"liked,omitempty"`
	Following bool      `json:"following,omitempty"`
}

// 投稿一覧API
func (app *App) getPostsAPI(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			page = parsed
		}
	}

	limit := 20
	offset := (page - 1) * limit

	posts := app.getTimelinePostsPaginated(userID, limit, offset)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Posts:   posts,
	})
}

// いいね機能API
func (app *App) likePostAPI(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Message: "Invalid post ID",
		})
		return
	}

	userID := r.Context().Value("user_id").(int)

	// 既にいいねしているかチェック
	var count int
	app.db.QueryRow("SELECT COUNT(*) FROM likes WHERE user_id = ? AND post_id = ?", userID, postID).Scan(&count)

	if count > 0 {
		// いいね解除
		app.db.Exec("DELETE FROM likes WHERE user_id = ? AND post_id = ?", userID, postID)
		app.db.Exec("UPDATE posts SET likes = likes - 1 WHERE id = ?", postID)
	} else {
		// いいね追加
		app.db.Exec("INSERT INTO likes (user_id, post_id) VALUES (?, ?)", userID, postID)
		app.db.Exec("UPDATE posts SET likes = likes + 1 WHERE id = ?", postID)
	}

	// 最新のいいね数取得
	var likes int
	app.db.QueryRow("SELECT likes FROM posts WHERE id = ?", postID).Scan(&likes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Likes:   likes,
		Liked:   count == 0,
	})
}

// コメント取得API
func (app *App) getCommentsAPI(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Message: "Invalid post ID",
		})
		return
	}

	comments := app.getPostComments(postID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success:  true,
		Comments: comments,
	})
}

// コメント作成API
func (app *App) createCommentAPI(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Message: "Invalid post ID",
		})
		return
	}

	userID := r.Context().Value("user_id").(int)

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// コメント作成
	_, err = app.db.Exec("INSERT INTO comments (user_id, post_id, content) VALUES (?, ?, ?)",
		userID, postID, req.Content)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Message: "Failed to create comment",
		})
		return
	}

	// 投稿のコメント数更新
	app.db.Exec("UPDATE posts SET comments = comments + 1 WHERE id = ?", postID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Message: "Comment created successfully",
	})
}

// 投稿削除API
func (app *App) deletePostAPI(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Message: "Invalid post ID",
		})
		return
	}

	userID := r.Context().Value("user_id").(int)

	// 投稿の所有者確認
	var ownerID int
	err = app.db.QueryRow("SELECT user_id FROM posts WHERE id = ?", postID).Scan(&ownerID)
	if err != nil || ownerID != userID {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	// 投稿削除（CASCADE制約で関連データも削除される）
	_, err = app.db.Exec("DELETE FROM posts WHERE id = ?", postID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Message: "Failed to delete post",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Message: "Post deleted successfully",
	})
}

// フォロー機能API
func (app *App) followUserAPI(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetUserID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	userID := r.Context().Value("user_id").(int)

	// 自分自身をフォローできない
	if userID == targetUserID {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Message: "Cannot follow yourself",
		})
		return
	}

	// 既にフォローしているかチェック
	var count int
	app.db.QueryRow("SELECT COUNT(*) FROM follows WHERE follower_id = ? AND following_id = ?", 
		userID, targetUserID).Scan(&count)

	if count > 0 {
		// フォロー解除
		app.db.Exec("DELETE FROM follows WHERE follower_id = ? AND following_id = ?", 
			userID, targetUserID)
	} else {
		// フォロー追加
		app.db.Exec("INSERT INTO follows (follower_id, following_id) VALUES (?, ?)", 
			userID, targetUserID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success:   true,
		Following: count == 0,
	})
}

// データベースクエリ関数群

// タイムライン投稿取得
func (app *App) getTimelinePosts(userID int) []Post {
	return app.getTimelinePostsPaginated(userID, 20, 0)
}

func (app *App) getTimelinePostsPaginated(userID int, limit, offset int) []Post {
	query := `
		SELECT p.id, p.user_id, u.username, u.avatar, p.content, p.image_url, 
		       p.likes, p.comments, p.created_at
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.user_id = ? OR p.user_id IN (
			SELECT following_id FROM follows WHERE follower_id = ?
		)
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?
	`
	return app.queryPosts(query, userID, userID, limit, offset)
}

// 最新投稿取得（未認証ユーザー向け）
func (app *App) getLatestPosts(limit int) []Post {
	query := `
		SELECT p.id, p.user_id, u.username, u.avatar, p.content, p.image_url, 
		       p.likes, p.comments, p.created_at
		FROM posts p
		JOIN users u ON p.user_id = u.id
		ORDER BY p.created_at DESC
		LIMIT ?
	`
	return app.queryPosts(query, limit)
}

// ユーザーの投稿取得
func (app *App) getUserPosts(userID, limit int) []Post {
	query := `
		SELECT p.id, p.user_id, u.username, u.avatar, p.content, p.image_url, 
		       p.likes, p.comments, p.created_at
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.user_id = ?
		ORDER BY p.created_at DESC
		LIMIT ?
	`
	return app.queryPosts(query, userID, limit)
}

// 投稿クエリ実行
func (app *App) queryPosts(query string, args ...interface{}) []Post {
	rows, err := app.db.Query(query, args...)
	if err != nil {
		return []Post{}
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.ID, &post.UserID, &post.Username, &post.Avatar, 
			&post.Content, &post.ImageURL, &post.Likes, &post.Comments, &post.CreatedAt)
		if err != nil {
			continue
		}
		posts = append(posts, post)
	}
	return posts
}

// コメント取得
func (app *App) getPostComments(postID int) []Comment {
	query := `
		SELECT c.id, c.user_id, c.post_id, u.username, u.avatar, c.content, c.created_at
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.post_id = ?
		ORDER BY c.created_at ASC
	`
	
	rows, err := app.db.Query(query, postID)
	if err != nil {
		return []Comment{}
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.ID, &comment.UserID, &comment.PostID, 
			&comment.Username, &comment.Avatar, &comment.Content, &comment.CreatedAt)
		if err != nil {
			continue
		}
		comments = append(comments, comment)
	}
	return comments
}

// おすすめユーザー取得
func (app *App) getSuggestedUsers(userID, limit int) []User {
	query := `
		SELECT id, username, avatar, bio
		FROM users
		WHERE id != ? AND id NOT IN (
			SELECT following_id FROM follows WHERE follower_id = ?
		)
		ORDER BY created_at DESC
		LIMIT ?
	`
	
	rows, err := app.db.Query(query, userID, userID, limit)
	if err != nil {
		return []User{}
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Avatar, &user.Bio)
		if err != nil {
			continue
		}
		users = append(users, user)
	}
	return users
}