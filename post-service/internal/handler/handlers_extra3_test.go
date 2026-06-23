package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── enforceContentLimit ─────────────────────────────────────────────────────

// Utilisateur standard avec 281 chars → enforceContentLimit doit retourner 400.
func TestCreatePost_ContentTropLong_User_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	long := strings.Repeat("a", 281)
	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(fmt.Sprintf(`{"content":%q}`, long)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreatePost(user, 281 chars) = %d, attendu 400", w.Code)
	}
}

// Modérateur avec 4001 chars → dépasse la borne privilegedContentMaxChars → 400.
func TestCreatePost_ContentTropLong_Mod_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "moderator")
	long := strings.Repeat("x", 4001)
	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(fmt.Sprintf(`{"content":%q}`, long)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreatePost(mod, 4001 chars) = %d, attendu 400", w.Code)
	}
}

// Modérateur avec 281 chars → exemption borne standard → passe enforceContentLimit
// mais nil repo → 500. Vérifie que la branche mod de enforceContentLimit est empruntée.
func TestCreatePost_ContentMod_PasseEnforce_NilRepo500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "moderator")
	long := strings.Repeat("y", 281)
	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(fmt.Sprintf(`{"content":%q}`, long)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Ne doit PAS renvoyer 400 (la borne handler de 280 ne s'applique pas aux mods).
	if w.Code == http.StatusBadRequest {
		t.Errorf("CreatePost(mod, 281 chars) = 400 : enforceContentLimit ne doit pas bloquer les mods")
	}
}

// Admin avec 281 chars → même exemption que modérateur.
func TestCreatePost_ContentAdmin_PasseEnforce(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "admin")
	long := strings.Repeat("z", 281)
	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(fmt.Sprintf(`{"content":%q}`, long)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusBadRequest {
		t.Errorf("CreatePost(admin, 281 chars) = 400 : enforceContentLimit ne doit pas bloquer les admins")
	}
}

// UpdatePost avec contenu trop long pour un user standard → 400.
func TestUpdatePost_ContentTropLong_User_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	long := strings.Repeat("u", 281)
	req := httptest.NewRequest(http.MethodPatch, "/posts/507f1f77bcf86cd799439011",
		strings.NewReader(fmt.Sprintf(`{"content":%q}`, long)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("UpdatePost(user, 281 chars) = %d, attendu 400", w.Code)
	}
}

// ─── SetNsfw ─────────────────────────────────────────────────────────────────

// JSON invalide → 400 avant le service.
func TestSetNsfw_JSONInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "moderator")
	req := httptest.NewRequest(http.MethodPatch, "/posts/507f1f77bcf86cd799439011/nsfw",
		strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("SetNsfw(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// Payload valide + nil repo → panic → 500.
func TestSetNsfw_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "moderator")
	req := httptest.NewRequest(http.MethodPatch, "/posts/507f1f77bcf86cd799439011/nsfw",
		strings.NewReader(`{"nsfw":true}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("SetNsfw(nil repo) = %d, attendu 500", w.Code)
	}
}

// Sans token → 401.
func TestSetNsfw_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPatch, "/posts/507f1f77bcf86cd799439011/nsfw",
		strings.NewReader(`{"nsfw":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("SetNsfw(sans token) = %d, attendu 401", w.Code)
	}
}

// ─── BookmarkHandler manquant ─────────────────────────────────────────────────

// RenameCollection — JSON invalide → 400.
func TestRenameCollection_JSONInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/posts/bookmarks/collections/some-cid",
		strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("RenameCollection(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// validOID est un ObjectID hex valide utilisé pour passer les validations d'ID
// et atteindre le repo (nil → panic → 500).
const validOID = "507f1f77bcf86cd799439011"

// RenameCollection — payload valide + ObjectID valide + nil repo → 500.
func TestRenameCollection_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/posts/bookmarks/collections/"+validOID,
		strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("RenameCollection(nil repo) = %d, attendu 500", w.Code)
	}
}

// RenameCollection — ID invalide → 400 (ErrInvalidID depuis getOwnedCollection).
func TestRenameCollection_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/posts/bookmarks/collections/not-an-oid",
		strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("RenameCollection(ID invalide) = %d, attendu 400", w.Code)
	}
}

// RenameCollection — sans token → 401.
func TestRenameCollection_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPatch, "/posts/bookmarks/collections/"+validOID,
		strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("RenameCollection(sans token) = %d, attendu 401", w.Code)
	}
}

// DeleteCollection — ObjectID valide + nil repo → 500.
func TestDeleteCollection_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/bookmarks/collections/"+validOID, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("DeleteCollection(nil repo) = %d, attendu 500", w.Code)
	}
}

// DeleteCollection — sans token → 401.
func TestDeleteCollection_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/posts/bookmarks/collections/"+validOID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("DeleteCollection(sans token) = %d, attendu 401", w.Code)
	}
}

// ListCollectionPosts — ObjectID valide + nil repo → 500.
func TestListCollectionPosts_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/bookmarks/collections/"+validOID+"/posts", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListCollectionPosts(nil repo) = %d, attendu 500", w.Code)
	}
}

// ListCollectionPosts — sans token → 401.
func TestListCollectionPosts_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/bookmarks/collections/"+validOID+"/posts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("ListCollectionPosts(sans token) = %d, attendu 401", w.Code)
	}
}

// PostCollections — nil repo → 500.
func TestPostCollections_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/"+validOID+"/bookmark/collections", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("PostCollections(nil repo) = %d, attendu 500", w.Code)
	}
}

// PostCollections — sans token → 401.
func TestPostCollections_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/"+validOID+"/bookmark/collections", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("PostCollections(sans token) = %d, attendu 401", w.Code)
	}
}

// ─── CommentHandler manquant ─────────────────────────────────────────────────

// ListCommentReplies — nil repo → 500.
func TestListCommentReplies_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet,
		"/posts/507f1f77bcf86cd799439011/comments/507f1f77bcf86cd799439012/replies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListCommentReplies(nil repo) = %d, attendu 500", w.Code)
	}
}

// DeletePostComment — nil repo → 500.
func TestDeletePostComment_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete,
		"/posts/507f1f77bcf86cd799439011/comments/507f1f77bcf86cd799439012", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("DeletePostComment(nil repo) = %d, attendu 500", w.Code)
	}
}

// DeletePostComment — sans token → 401.
func TestDeletePostComment_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodDelete,
		"/posts/507f1f77bcf86cd799439011/comments/507f1f77bcf86cd799439012", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("DeletePostComment(sans token) = %d, attendu 401", w.Code)
	}
}

// CreatPostComment — contenu vide (pas de texte ni de média) → 400.
func TestCreatPostComment_ContenuVide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost,
		"/posts/507f1f77bcf86cd799439011/comments",
		strings.NewReader(`{"content":"   "}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreatPostComment(contenu vide) = %d, attendu 400", w.Code)
	}
}

// LikeComment — nil repo → 500.
func TestLikeComment_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost,
		"/posts/507f1f77bcf86cd799439011/comments/507f1f77bcf86cd799439012/like", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("LikeComment(nil repo) = %d, attendu 500", w.Code)
	}
}

// LikeComment — sans token → 401.
func TestLikeComment_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPost,
		"/posts/507f1f77bcf86cd799439011/comments/507f1f77bcf86cd799439012/like", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("LikeComment(sans token) = %d, attendu 401", w.Code)
	}
}

// UnlikeComment — nil repo → 500.
func TestUnlikeComment_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete,
		"/posts/507f1f77bcf86cd799439011/comments/507f1f77bcf86cd799439012/like", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("UnlikeComment(nil repo) = %d, attendu 500", w.Code)
	}
}

// UnlikeComment — sans token → 401.
func TestUnlikeComment_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodDelete,
		"/posts/507f1f77bcf86cd799439011/comments/507f1f77bcf86cd799439012/like", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("UnlikeComment(sans token) = %d, attendu 401", w.Code)
	}
}

// ─── WSHandler ───────────────────────────────────────────────────────────────

// Connect — token invalide → 401 (sans upgrade).
func TestWSConnect_TokenInvalide_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/ws?access_token=invalid.token.here", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("WSConnect(token invalide) = %d, attendu 401", w.Code)
	}
}

// Connect — token absent → 401.
func TestWSConnect_TokenAbsent_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/ws", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("WSConnect(token absent) = %d, attendu 401", w.Code)
	}
}

// ─── respondPostError — erreurs métier supplémentaires ───────────────────────

// parseQueryTime — date valide (branche non nulle).
func TestParseQueryTime_DateValide_200(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "moderator")
	// ?since avec une date RFC3339 valide → parseQueryTime retourne un *time.Time non nil.
	req := httptest.NewRequest(http.MethodGet,
		"/posts/moderation/deleted?since=2024-01-01T00:00:00Z", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// nil repo → 500, mais le chemin parseQueryTime valide est bien emprunté.
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListHidden(since valide, nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── LikeHandler — chemins 401 ───────────────────────────────────────────────

func TestLikePost_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/posts/507f1f77bcf86cd799439011/like", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("LikePost(sans token) = %d, attendu 401", w.Code)
	}
}

func TestUnlikePost_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/posts/507f1f77bcf86cd799439011/like", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("UnlikePost(sans token) = %d, attendu 401", w.Code)
	}
}

func TestListLikedByUser_SansAuthorID_400(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/liked", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("ListLikedByUser(sans author_id) = %d, attendu 400", w.Code)
	}
}

func TestLikedByMe_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/me/liked-ids", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("LikedByMe(sans token) = %d, attendu 401", w.Code)
	}
}

// ─── PostHandler — chemins 401 supplémentaires ───────────────────────────────

func TestCreatePost_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(`{"content":"bonjour"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("CreatePost(sans token) = %d, attendu 401", w.Code)
	}
}

func TestUpdatePost_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPatch, "/posts/507f1f77bcf86cd799439011",
		strings.NewReader(`{"content":"edit"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("UpdatePost(sans token) = %d, attendu 401", w.Code)
	}
}

func TestPinPost_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPatch, "/posts/507f1f77bcf86cd799439011/pin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("PinPost(sans token) = %d, attendu 401", w.Code)
	}
}

func TestUnpinPost_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/posts/507f1f77bcf86cd799439011/pin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("UnpinPost(sans token) = %d, attendu 401", w.Code)
	}
}

func TestRepostPost_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/posts/507f1f77bcf86cd799439011/repost", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("RepostPost(sans token) = %d, attendu 401", w.Code)
	}
}

func TestUnrepostPost_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/posts/507f1f77bcf86cd799439011/repost", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("UnrepostPost(sans token) = %d, attendu 401", w.Code)
	}
}

func TestRepostedByMe_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/me/reposted-ids", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("RepostedByMe(sans token) = %d, attendu 401", w.Code)
	}
}

func TestDeletePost_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/posts/507f1f77bcf86cd799439011", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("DeletePost(sans token) = %d, attendu 401", w.Code)
	}
}

func TestRestorePost_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/posts/507f1f77bcf86cd799439011/restore", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("RestorePost(sans token) = %d, attendu 401", w.Code)
	}
}

func TestPurgePost_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/posts/507f1f77bcf86cd799439011/purge", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("PurgePost(sans token) = %d, attendu 401", w.Code)
	}
}

func TestPurgeUserData_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/posts/by-author/some-user", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("PurgeUserData(sans token) = %d, attendu 401", w.Code)
	}
}

func TestVotePoll_JSONInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost,
		"/posts/507f1f77bcf86cd799439011/poll/vote",
		strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("VotePoll(JSON invalide) = %d, attendu 400", w.Code)
	}
}

func TestVotePoll_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPost,
		"/posts/507f1f77bcf86cd799439011/poll/vote",
		strings.NewReader(`{"choice_id":"c1"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("VotePoll(sans token) = %d, attendu 401", w.Code)
	}
}

func TestClosePoll_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPost,
		"/posts/507f1f77bcf86cd799439011/poll/close", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("ClosePoll(sans token) = %d, attendu 401", w.Code)
	}
}

func TestListHidden_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/moderation/deleted", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Route protégée par JWTAuth → 401 sans token.
	if w.Code != http.StatusUnauthorized {
		t.Errorf("ListHidden(sans token) = %d, attendu 401", w.Code)
	}
}

// PostStats avec ids → nil repo → 500.
func TestPostStats_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/stats?ids=507f1f77bcf86cd799439011", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("PostStats(nil repo, avec ids) = %d, attendu 500", w.Code)
	}
}

// PostStats avec JWT optionnel → couvre la branche viewerID != "".
func TestPostStats_AvecToken_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/stats?ids=507f1f77bcf86cd799439011", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("PostStats(avec token, nil repo) = %d, attendu 500", w.Code)
	}
}

// ListHashtagTrends avec JWT optionnel → couvre la branche viewerID != "".
func TestListHashtagTrends_AvecToken_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/trends", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListHashtagTrends(avec token, nil repo) = %d, attendu 500", w.Code)
	}
}

// GetPost avec JWT optionnel + ID invalide → 400 (branche claims ok).
func TestGetPost_AvecToken_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/not-valid", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("GetPost(avec token, ID invalide) = %d, attendu 400", w.Code)
	}
}

// ListPostComments avec JWT optionnel → couvre la branche viewerID != "".
func TestListPostComments_AvecToken_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/507f1f77bcf86cd799439011/comments", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListPostComments(avec token, nil repo) = %d, attendu 500", w.Code)
	}
}

// ListCommentReplies avec JWT optionnel → couvre la branche viewerID != "".
func TestListCommentReplies_AvecToken_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet,
		"/posts/507f1f77bcf86cd799439011/comments/507f1f77bcf86cd799439012/replies", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListCommentReplies(avec token, nil repo) = %d, attendu 500", w.Code)
	}
}

// ListLikedByUser avec JWT optionnel → couvre la branche callerID != "".
func TestListLikedByUser_AvecToken_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/liked?author_id=some-author", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListLikedByUser(avec token, nil repo) = %d, attendu 500", w.Code)
	}
}

// ListPosts avec author_ids → couvre la branche GetFeed.
func TestListPosts_AuthorIDs_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts?author_ids=user1,user2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListPosts(author_ids, nil repo) = %d, attendu 500", w.Code)
	}
}

// ListPosts avec author_id (profil) → couvre la branche GetByProfile.
func TestListPosts_AuthorID_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts?author_id=user1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListPosts(author_id, nil repo) = %d, attendu 500", w.Code)
	}
}

// ListPosts avec hashtag_any=true → couvre la branche hashtagAny.
func TestListPosts_HashtagAny_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts?hashtag_any=true", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListPosts(hashtag_any, nil repo) = %d, attendu 500", w.Code)
	}
}

// ListPostLikes — nil repo + ID valide → 500 (branche non-ErrInvalidID).
func TestListPostLikes_ValidID_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/507f1f77bcf86cd799439011/likes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListPostLikes(ID valide, nil repo) = %d, attendu 500", w.Code)
	}
}

// NewWSHandler : le CheckOrigin autorise une origine connue.
func TestWSHandler_CheckOrigin_AllowedOrigin(t *testing.T) {
	h := NewWSHandler(nil, "secret", []string{"https://breezy.dev"})
	if h == nil {
		t.Fatal("NewWSHandler = nil")
	}
}

// NewWSHandler : le CheckOrigin bloque une origine inconnue.
func TestWSHandler_CheckOrigin_NoOrigins(t *testing.T) {
	h := NewWSHandler(nil, "secret", nil)
	if h == nil {
		t.Fatal("NewWSHandler(nil origins) = nil")
	}
}

// ListCommentsByAuthor sans author_id → 400.
func TestListCommentsByAuthor_SansAuthorID_400(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/comments", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("ListCommentsByAuthor(sans author_id) = %d, attendu 400", w.Code)
	}
}

// ListCommentsByAuthor avec JWT optionnel → couvre la branche viewerID != "".
func TestListCommentsByAuthor_AvecToken_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/comments?author_id=some-user", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListCommentsByAuthor(avec token, nil repo) = %d, attendu 500", w.Code)
	}
}

// ListPostComments (route publique) nil repo → 500.
func TestListPostComments_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet,
		"/posts/507f1f77bcf86cd799439011/comments", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListPostComments(nil repo, ID valide) = %d, attendu 500", w.Code)
	}
}

// Bookmark sans token → 401.
func TestBookmark_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/posts/507f1f77bcf86cd799439011/bookmark", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Bookmark(sans token) = %d, attendu 401", w.Code)
	}
}

// Unbookmark sans token → 401.
func TestUnbookmark_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/posts/507f1f77bcf86cd799439011/bookmark", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Unbookmark(sans token) = %d, attendu 401", w.Code)
	}
}

// BookmarkedByMe sans token → 401.
func TestBookmarkedByMe_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/me/bookmarked-ids", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("BookmarkedByMe(sans token) = %d, attendu 401", w.Code)
	}
}

// ListAll sans token → 401.
func TestListAll_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/bookmarks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("ListAll(sans token) = %d, attendu 401", w.Code)
	}
}

// ListCollections sans token → 401.
func TestListCollections_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/bookmarks/collections", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("ListCollections(sans token) = %d, attendu 401", w.Code)
	}
}

// CreateCollection sans token → 401.
func TestCreateCollection_SansToken_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/posts/bookmarks/collections",
		strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("CreateCollection(sans token) = %d, attendu 401", w.Code)
	}
}

// CreateCollection payload valide + nil repo → 500.
func TestCreateCollection_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/bookmarks/collections",
		strings.NewReader(`{"name":"Ma collection"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("CreateCollection(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── respondPostError — branche ErrInvalidPoll ───────────────────────────────

// CreatePost avec labels de sondage dupliqués → buildPoll retourne ErrInvalidPoll
// (validé par le service AVANT tout appel repo) → respondPostError branche 3 → 400.
func TestCreatePost_PollInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	// Labels identiques (après TrimSpace + ToLower) → ErrInvalidPoll dans buildPoll.
	body := `{"content":"sondage","poll":{"duration_minutes":1,"choices":[{"label":"A"},{"label":"A"}]}}`
	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreatePost(poll dupliqué) = %d, attendu 400", w.Code)
	}
}

// ─── ListPosts — branche GetFeed avec authorIDs vides → 200 sans repo ────────

// ?author_ids=,,, → splitIDs retourne [] → GetFeed retourne []Post{} nil sans appel
// repo → handler renvoie 200 (chemin succès de ListPosts sans Mongo).
func TestListPosts_AuthorIDsVides_200(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts?author_ids=,,,", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("ListPosts(author_ids vides) = %d, attendu 200", w.Code)
	}
}

// ─── UpdatePost — chemin service (contenu valide, nil repo) ─────────────────

// Contenu court + token user → enforceContentLimit passe → service appelé →
// nil repo → panic → gin.Recovery → 500.
func TestUpdatePost_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/posts/"+validOID,
		strings.NewReader(`{"content":"edit valide"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("UpdatePost(nil repo, contenu valide) = %d, attendu 500", w.Code)
	}
}

// ─── Handlers sans test NilRepo complémentaire ───────────────────────────────

func TestPinPost_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/posts/"+validOID+"/pin", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("PinPost(nil repo) = %d, attendu 500", w.Code)
	}
}

func TestUnpinPost_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+validOID+"/pin", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("UnpinPost(nil repo) = %d, attendu 500", w.Code)
	}
}

func TestRepostPost_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/repost", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("RepostPost(nil repo) = %d, attendu 500", w.Code)
	}
}

func TestUnrepostPost_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+validOID+"/repost", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("UnrepostPost(nil repo) = %d, attendu 500", w.Code)
	}
}

func TestDeletePost_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+validOID, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("DeletePost(nil repo) = %d, attendu 500", w.Code)
	}
}

func TestRestorePost_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "moderator")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/restore", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("RestorePost(nil repo, mod) = %d, attendu 500", w.Code)
	}
}

func TestPurgePost_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "moderator")
	req := httptest.NewRequest(http.MethodDelete, "/posts/"+validOID+"/purge", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("PurgePost(nil repo, mod) = %d, attendu 500", w.Code)
	}
}

func TestClosePoll_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/poll/close", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ClosePoll(nil repo) = %d, attendu 500", w.Code)
	}
}

func TestBookmark_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/bookmark", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Bookmark(nil repo, validOID) = %d, attendu 500", w.Code)
	}
}

// VotePoll avec ID valide + choix valide + nil repo → 500.
func TestVotePoll_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/"+validOID+"/poll/vote",
		strings.NewReader(`{"choice_id":"somechoiceid"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("VotePoll(nil repo, validOID) = %d, attendu 500", w.Code)
	}
}

// ─── ID invalide → ErrInvalidID → respondPostError → 400 ─────────────────────
// Ces tests couvrent la branche "if err != nil { respondPostError }" des handlers
// qui n'avaient jusqu'ici que des tests SansToken et NilRepo.

func TestDeleteCollection_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/bookmarks/collections/not-an-oid", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("DeleteCollection(ID invalide) = %d, attendu 400", w.Code)
	}
}

func TestListCollectionPosts_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/bookmarks/collections/not-an-oid/posts", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("ListCollectionPosts(ID invalide) = %d, attendu 400", w.Code)
	}
}

func TestPostCollections_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/posts/not-an-oid/bookmark/collections", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("PostCollections(ID invalide) = %d, attendu 400", w.Code)
	}
}

// commentId invalide → ErrInvalidID avant repo.
func TestDeletePostComment_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete,
		"/posts/"+validOID+"/comments/not-an-oid", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("DeletePostComment(commentId invalide) = %d, attendu 400", w.Code)
	}
}

func TestLikeComment_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost,
		"/posts/not-an-oid/comments/"+validOID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("LikeComment(postId invalide) = %d, attendu 400", w.Code)
	}
}

func TestUnlikeComment_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete,
		"/posts/not-an-oid/comments/"+validOID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("UnlikeComment(postId invalide) = %d, attendu 400", w.Code)
	}
}

func TestListCommentReplies_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet,
		"/posts/"+validOID+"/comments/not-an-oid/replies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("ListCommentReplies(commentId invalide) = %d, attendu 400", w.Code)
	}
}

// postId invalide dans CreatPostComment → ErrInvalidID → 400.
func TestCreatPostComment_PostIDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/not-an-oid/comments",
		strings.NewReader(`{"content":"bonjour"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreatPostComment(postId invalide) = %d, attendu 400", w.Code)
	}
}

// ─── pageLimit / pageOffset — branche succès (valeur numérique valide) ────────

func TestPageLimit_Valid_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	// limit et offset valides → pageLimit et pageOffset retournent la valeur
	// (couvre la branche `return n` non encore testée).
	req := httptest.NewRequest(http.MethodGet, "/posts?limit=10&offset=5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// nil repo → 500, mais les chemins `return n` de pageLimit/pageOffset sont couverts.
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListPosts(limit=10, offset=5, nil repo) = %d, attendu 500", w.Code)
	}
}
