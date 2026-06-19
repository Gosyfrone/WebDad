package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── RepostPost / UnrepostPost / RepostedByMe ─────────────────────────────────

func TestRepostPost_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/not-valid/repost", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("RepostPost(ID invalide) = %d, attendu 400", w.Code)
	}
}

func TestUnrepostPost_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/not-valid/repost", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("UnrepostPost(ID invalide) = %d, attendu 400", w.Code)
	}
}

func TestRepostedByMe_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/me/reposted-ids", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("RepostedByMe(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── ListHashtagTrends — nil repo → 500 ──────────────────────────────────────

func TestListHashtagTrends_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/trends", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListHashtagTrends(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── PurgeUserData — admin token, nil repo → 500 ────────────────────────────

func TestPurgeUserData_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "admin")
	req := httptest.NewRequest(http.MethodDelete, "/posts/by-author/some-author", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("PurgeUserData(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── LikeHandler — ID invalide → 400 ─────────────────────────────────────────

func TestLikePost_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/not-valid/like", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("LikePost(ID invalide) = %d, attendu 400", w.Code)
	}
}

func TestUnlikePost_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/not-valid/like", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("UnlikePost(ID invalide) = %d, attendu 400", w.Code)
	}
}

func TestListPostLikes_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/not-valid/likes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("ListPostLikes(ID invalide) = %d, attendu 400", w.Code)
	}
}

func TestListLikedByUser_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	// author_id requis, sinon le handler retourne 400 avant le repo
	req := httptest.NewRequest(http.MethodGet, "/posts/liked?author_id=some-author", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListLikedByUser(nil repo) = %d, attendu 500", w.Code)
	}
}

func TestLikedByMe_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/me/liked-ids", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("LikedByMe(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── BookmarkHandler — nil repo → 500 ────────────────────────────────────────

func TestListCollections_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/bookmarks/collections", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListCollections(nil repo) = %d, attendu 500", w.Code)
	}
}

func TestCreateCollection_JSONInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/bookmarks/collections", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateCollection(JSON invalide) = %d, attendu 400", w.Code)
	}
}

func TestListAll_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/bookmarks", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListAll(nil repo) = %d, attendu 500", w.Code)
	}
}

func TestBookmark_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/not-valid/bookmark", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Bookmark(ID invalide) = %d, attendu 400", w.Code)
	}
}

func TestUnbookmark_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	// Unbookmark va direct au repo sans valider l'ID → nil repo → panic → 500
	req := httptest.NewRequest(http.MethodDelete, "/posts/not-valid/bookmark", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Unbookmark(nil repo) = %d, attendu 500", w.Code)
	}
}

func TestBookmarkedByMe_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/me/bookmarked-ids", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("BookmarkedByMe(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── CommentHandler — ID invalide → 400 / nil repo → 500 ─────────────────────

func TestListPostComments_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/not-valid/comments", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("ListPostComments(ID invalide) = %d, attendu 400", w.Code)
	}
}

func TestCreatPostComment_JSONInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/507f1f77bcf86cd799439011/comments", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreatPostComment(JSON invalide) = %d, attendu 400", w.Code)
	}
}

func TestListCommentsByAuthor_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	// author_id requis, sinon 400 avant le repo
	req := httptest.NewRequest(http.MethodGet, "/posts/comments?author_id=some-user", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListCommentsByAuthor(nil repo) = %d, attendu 500", w.Code)
	}
}

// CommentStats avec ids vide → 200 early return (pas d'appel repo).
func TestCommentStats_IdsVides_200(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/comments/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("CommentStats(ids vides) = %d, attendu 200", w.Code)
	}
}

// CommentStats avec ids → nil repo → panic → 500.
func TestCommentStats_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/comments/stats?ids=507f1f77bcf86cd799439011", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("CommentStats(nil repo) = %d, attendu 500", w.Code)
	}
}
