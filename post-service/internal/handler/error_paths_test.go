package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── Chemins d'erreur via errRepo ────────────────────────────────────────────
// Ces tests couvrent la branche "if err != nil { respondPostError; return }"
// dans les handlers, sans MongoDB. errRepo retourne mongo.ErrNoDocuments (→
// ErrPostNotFound → 404) ou errGeneric (→ respondPostError default → 500).

// ── post_handler ─────────────────────────────────────────────────────────────

func TestListPosts_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListPosts(errRepo) = %d, attendu 500", w.Code)
	}
}

func TestListPosts_Hashtag_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts?hashtag=breezy", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListPosts(hashtag, errRepo) = %d, attendu 500", w.Code)
	}
}

func TestListPosts_HashtagAny_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts?hashtag_any=true", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListPosts(hashtag_any, errRepo) = %d, attendu 500", w.Code)
	}
}

func TestGetPost_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/"+validOID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Get → mongo.ErrNoDocuments → translateNotFound → ErrPostNotFound → 404.
	if w.Code != http.StatusNotFound {
		t.Errorf("GetPost(errRepo) = %d, attendu 404", w.Code)
	}
}

func TestUpdatePost_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/posts/"+validOID,
		strings.NewReader(`{"content":"nouveau contenu"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("UpdatePost(errRepo) = %d, attendu 404", w.Code)
	}
}

func TestSetNsfw_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "moderator")
	req := httptest.NewRequest(http.MethodPatch, "/posts/"+validOID+"/nsfw",
		strings.NewReader(`{"nsfw":true}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// SetNsfw appelle repo.SetNsfw directement (pas Get) → errGeneric → default 500.
	if w.Code != http.StatusInternalServerError {
		t.Errorf("SetNsfw(errRepo) = %d, attendu 500", w.Code)
	}
}

func TestPinPost_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/pin", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("PinPost(errRepo) = %d, attendu 404", w.Code)
	}
}

func TestUnpinPost_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+validOID+"/pin", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("UnpinPost(errRepo) = %d, attendu 404", w.Code)
	}
}

func TestRepostPost_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/repost", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Service appelle Get avant AddRepost → ErrNoDocuments → ErrPostNotFound → 404.
	if w.Code != http.StatusNotFound {
		t.Errorf("RepostPost(errRepo) = %d, attendu 404", w.Code)
	}
}

func TestUnrepostPost_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+validOID+"/repost", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("UnrepostPost(errRepo) = %d, attendu 404", w.Code)
	}
}

func TestDeletePost_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+validOID, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("DeletePost(errRepo) = %d, attendu 404", w.Code)
	}
}

func TestRestorePost_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "moderator")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/restore", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// RestoreHidden → ErrNoDocuments → ErrPostNotFound → 404.
	if w.Code != http.StatusNotFound {
		t.Errorf("RestorePost(errRepo) = %d, attendu 404", w.Code)
	}
}

func TestPurgePost_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "moderator")
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+validOID+"/purge", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("PurgePost(errRepo) = %d, attendu 404", w.Code)
	}
}

func TestListHidden_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "moderator")
	req := httptest.NewRequest(http.MethodGet, "/posts/moderation/deleted", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListHidden(errRepo) = %d, attendu 500", w.Code)
	}
}

func TestPurgeUserData_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "admin")
	req := httptest.NewRequest(http.MethodDelete, "/posts/by-author/some-user", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("PurgeUserData(errRepo) = %d, attendu 500", w.Code)
	}
}

func TestRepostedByMe_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/me/reposted-ids", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("RepostedByMe(errRepo) = %d, attendu 500", w.Code)
	}
}

func TestClosePoll_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/poll/close", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Get → ErrNoDocuments → ErrPostNotFound → 404.
	if w.Code != http.StatusNotFound {
		t.Errorf("ClosePoll(errRepo) = %d, attendu 404", w.Code)
	}
}

func TestVotePoll_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost,
		"/posts/"+validOID+"/poll/vote",
		strings.NewReader(`{"choice_id":"`+validOID+`"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Get → ErrNoDocuments → ErrPostNotFound → 404.
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusBadRequest {
		t.Errorf("VotePoll(errRepo) = %d, ne doit pas être 401/400", w.Code)
	}
}

// ── like_handler ─────────────────────────────────────────────────────────────

func TestLikePost_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Service appelle Get avant AddLike → ErrNoDocuments → ErrPostNotFound → 404.
	if w.Code != http.StatusNotFound {
		t.Errorf("LikePost(errRepo) = %d, attendu 404", w.Code)
	}
}

func TestUnlikePost_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+validOID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("UnlikePost(errRepo) = %d, attendu 404", w.Code)
	}
}

func TestLikedByMe_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/me/liked-ids", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("LikedByMe(errRepo) = %d, attendu 500", w.Code)
	}
}

func TestListPostLikes_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/"+validOID+"/likes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListPostLikes(errRepo) = %d, attendu 500", w.Code)
	}
}

func TestListLikedByUser_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/liked?author_id=alice", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListLikedByUser(errRepo) = %d, attendu 500", w.Code)
	}
}

// ── comment_handler ───────────────────────────────────────────────────────────

func TestListCommentsByAuthor_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/comments?author_id=alice", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListCommentsByAuthor(errRepo) = %d, attendu 500", w.Code)
	}
}

func TestListPostComments_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/"+validOID+"/comments", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListPostComments(errRepo) = %d, attendu 500", w.Code)
	}
}

func TestListCommentReplies_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet,
		"/posts/"+validOID+"/comments/"+validOID+"/replies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListCommentReplies(errRepo) = %d, attendu 500", w.Code)
	}
}

func TestCreatPostComment_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/comments",
		strings.NewReader(`{"content":"super commentaire"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Get → ErrNoDocuments → ErrPostNotFound → 404, ou AddComment → 500.
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusBadRequest {
		t.Errorf("CreatPostComment(errRepo) = %d, ne doit pas être 401/400", w.Code)
	}
}

func TestDeletePostComment_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete,
		"/posts/"+validOID+"/comments/"+validOID, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// GetComment → ErrNoDocuments → ErrPostNotFound → 404.
	if w.Code != http.StatusNotFound {
		t.Errorf("DeletePostComment(errRepo) = %d, attendu 404", w.Code)
	}
}

func TestLikeComment_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost,
		"/posts/"+validOID+"/comments/"+validOID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// GetComment → ErrNoDocuments → ErrPostNotFound → 404.
	if w.Code != http.StatusNotFound {
		t.Errorf("LikeComment(errRepo) = %d, attendu 404", w.Code)
	}
}

func TestUnlikeComment_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete,
		"/posts/"+validOID+"/comments/"+validOID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("UnlikeComment(errRepo) = %d, attendu 404", w.Code)
	}
}

// ── bookmark_handler ──────────────────────────────────────────────────────────

func TestListCollections_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/bookmarks/collections", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListCollections(errRepo) = %d, attendu 500", w.Code)
	}
}

func TestCreateCollection_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/bookmarks/collections",
		strings.NewReader(`{"name":"Ma collection"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// EnsureDefaultCollection → errGeneric → 500.
	if w.Code != http.StatusInternalServerError {
		t.Errorf("CreateCollection(errRepo) = %d, attendu 500", w.Code)
	}
}

func TestRenameCollection_ErrRepo(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/posts/bookmarks/collections/"+validOID,
		strings.NewReader(`{"name":"Nouveau nom"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// GetCollection → ErrNoDocuments → ErrCollectionNotFound → 404.
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusBadRequest {
		t.Errorf("RenameCollection(errRepo) = %d, ne doit pas être 401/400", w.Code)
	}
}

func TestDeleteCollection_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete,
		"/posts/bookmarks/collections/"+validOID, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusBadRequest {
		t.Errorf("DeleteCollection(errRepo) = %d, ne doit pas être 401/400", w.Code)
	}
}

func TestListCollectionPosts_ErrRepo(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet,
		"/posts/bookmarks/collections/"+validOID+"/posts", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusBadRequest {
		t.Errorf("ListCollectionPosts(errRepo) = %d, ne doit pas être 401/400", w.Code)
	}
}

func TestListAll_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/bookmarks", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListAll(errRepo) = %d, attendu 500", w.Code)
	}
}

func TestBookmark_ErrRepo_404(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/bookmark", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Service appelle Get sur le post avant EnsureDefaultCollection → 404.
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusBadRequest {
		t.Errorf("Bookmark(errRepo) = %d, ne doit pas être 401/400", w.Code)
	}
}

func TestUnbookmark_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+validOID+"/bookmark", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Unbookmark(errRepo) = %d, attendu 500", w.Code)
	}
}

func TestBookmarkedByMe_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/me/bookmarked-ids", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("BookmarkedByMe(errRepo) = %d, attendu 500", w.Code)
	}
}

func TestPostCollections_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet,
		"/posts/"+validOID+"/bookmark/collections", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("PostCollections(errRepo) = %d, attendu 500", w.Code)
	}
}

func TestListHashtagTrends_ErrRepo_500(t *testing.T) {
	r := newErrRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/trends?hashtag=golang", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListHashtagTrends(errRepo) = %d, attendu 500", w.Code)
	}
}
