document.addEventListener('DOMContentLoaded', function() {
    // いいねボタンの処理
    document.addEventListener('click', function(e) {
        if (e.target.classList.contains('like-btn') || e.target.closest('.like-btn')) {
            e.preventDefault();
            const btn = e.target.closest('.like-btn');
            const postId = btn.dataset.postId;
            
            fetch(`/api/posts/${postId}/like`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    const likeCount = btn.querySelector('.like-count');
                    likeCount.textContent = data.likes;
                    btn.style.color = data.liked ? '#e91e63' : '#657786';
                }
            })
            .catch(error => console.error('Error:', error));
        }
    });

    // コメントボタンの処理
    document.addEventListener('click', function(e) {
        if (e.target.classList.contains('comment-btn') || e.target.closest('.comment-btn')) {
            e.preventDefault();
            const btn = e.target.closest('.comment-btn');
            const postId = btn.dataset.postId;
            const commentsDiv = document.getElementById(`comments-${postId}`);
            
            if (commentsDiv.style.display === 'none') {
                commentsDiv.style.display = 'block';
                loadComments(postId);
            } else {
                commentsDiv.style.display = 'none';
            }
        }
    });

    // コメント送信
    document.addEventListener('submit', function(e) {
        if (e.target.classList.contains('comment-submit')) {
            e.preventDefault();
            const form = e.target;
            const postId = form.dataset.postId;
            const content = form.querySelector('input[name="content"]').value;
            
            fetch(`/api/posts/${postId}/comments`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ content: content })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    form.querySelector('input[name="content"]').value = '';
                    loadComments(postId);
                    // コメント数を更新
                    const commentBtn = document.querySelector(`[data-post-id="${postId}"].comment-btn`);
                    const commentCount = commentBtn.querySelector('.comment-count');
                    commentCount.textContent = parseInt(commentCount.textContent) + 1;
                }
            })
            .catch(error => console.error('Error:', error));
        }
    });

    // 投稿削除
    document.addEventListener('click', function(e) {
        if (e.target.classList.contains('delete-btn')) {
            e.preventDefault();
            const postId = e.target.dataset.postId;
            
            if (confirm('この投稿を削除しますか？')) {
                fetch(`/api/posts/${postId}`, {
                    method: 'DELETE'
                })
                .then(response => response.json())
                .then(data => {
                    if (data.success) {
                        const postElement = document.querySelector(`[data-post-id="${postId}"]`);
                        postElement.remove();
                    }
                })
                .catch(error => console.error('Error:', error));
            }
        }
    });

    // フォローボタンの処理
    document.addEventListener('click', function(e) {
        if (e.target.classList.contains('follow-btn')) {
            e.preventDefault();
            const btn = e.target;
            const userId = btn.dataset.userId;
            
            fetch(`/api/users/${userId}/follow`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    btn.textContent = data.following ? 'フォロー解除' : 'フォロー';
                    btn.classList.toggle('btn-primary', !data.following);
                    btn.classList.toggle('btn-secondary', data.following);
                }
            })
            .catch(error => console.error('Error:', error));
        }
    });

    // 投稿フォーム送信時の画像プレビュー
    const imageInput = document.querySelector('input[type="file"][name="image"]');
    if (imageInput) {
        imageInput.addEventListener('change', function(e) {
            const file = e.target.files[0];
            if (file) {
                const reader = new FileReader();
                reader.onload = function(e) {
                    let preview = document.getElementById('image-preview');
                    if (!preview) {
                        preview = document.createElement('div');
                        preview.id = 'image-preview';
                        preview.innerHTML = '<img id="preview-img" style="max-width: 200px; border-radius: 8px; margin-top: 1rem;"><button type="button" id="remove-preview" style="margin-left: 1rem;">削除</button>';
                        imageInput.parentNode.appendChild(preview);
                    }
                    document.getElementById('preview-img').src = e.target.result;
                };
                reader.readAsDataURL(file);
            }
        });

        document.addEventListener('click', function(e) {
            if (e.target.id === 'remove-preview') {
                imageInput.value = '';
                document.getElementById('image-preview').remove();
            }
        });
    }
});

// コメント読み込み
function loadComments(postId) {
    fetch(`/api/posts/${postId}/comments`)
        .then(response => response.json())
        .then(data => {
            const commentList = document.getElementById(`comment-list-${postId}`);
            commentList.innerHTML = '';
            
            data.comments.forEach(comment => {
                const commentDiv = document.createElement('div');
                commentDiv.className = 'comment';
                commentDiv.innerHTML = `
                    <div style="display: flex; align-items: center; margin-bottom: 0.5rem;">
                        <img src="${comment.avatar}" alt="${comment.username}" class="avatar-sm" style="margin-right: 0.5rem;">
                        <strong>${comment.username}</strong>
                        <span style="color: #657786; font-size: 0.875rem; margin-left: 0.5rem;">
                            ${new Date(comment.created_at).toLocaleString('ja-JP')}
                        </span>
                    </div>
                    <p style="margin-left: 2.5rem;">${comment.content}</p>
                `;
                commentList.appendChild(commentDiv);
            });
        })
        .catch(error => console.error('Error:', error));
}

// 無限スクロール（オプション）
let loading = false;
let page = 1;

window.addEventListener('scroll', function() {
    if (loading) return;
    
    if (window.innerHeight + window.scrollY >= document.body.offsetHeight - 1000) {
        loading = true;
        loadMorePosts();
    }
});

function loadMorePosts() {
    page++;
    fetch(`/api/posts?page=${page}`)
        .then(response => response.json())
        .then(data => {
            if (data.posts && data.posts.length > 0) {
                const postsContainer = document.querySelector('.posts');
                data.posts.forEach(post => {
                    const postDiv = createPostElement(post);
                    postsContainer.appendChild(postDiv);
                });
            }
            loading = false;
        })
        .catch(error => {
            console.error('Error:', error);
            loading = false;
        });
}

function createPostElement(post) {
    const postDiv = document.createElement('div');
    postDiv.className = 'post';
    postDiv.dataset.postId = post.id;
    
    postDiv.innerHTML = `
        <div class="post-header">
            <img src="${post.avatar}" alt="${post.username}" class="avatar">
            <div class="post-info">
                <strong>${post.username}</strong>
                <span class="post-time">${new Date(post.created_at).toLocaleString('ja-JP')}</span>
            </div>
        </div>
        <div class="post-content">
            <p>${post.content}</p>
            ${post.image_url ? `<img src="${post.image_url}" alt="投稿画像" class="post-image">` : ''}
        </div>
        <div class="post-actions">
            <button class="btn btn-sm like-btn" data-post-id="${post.id}">
                ❤️ <span class="like-count">${post.likes}</span>
            </button>
            <button class="btn btn-sm comment-btn" data-post-id="${post.id}">
                💬 <span class="comment-count">${post.comments}</span>
            </button>
        </div>
    `;
    
    return postDiv;
}