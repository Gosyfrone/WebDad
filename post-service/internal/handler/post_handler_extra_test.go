package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/webdad/post-service/internal/middleware"
	"github.com/webdad/post-service/internal/realtime"
	"github.com/webdad/post-service/internal/service"
)

const (
	nilTestSecret   = "nil-test-secret"
	nilInternalSec  = "nil-internal-secret"
)

// newNilRepoRouter monte les routes avec un PostService à repo nil + gin.Recovery().
// Les panics (nil repo) sont converties en 500 ; les gardes de service renvoient
// leurs erreurs métier AVANT d'atteindre le repo.
func newNilRepoRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	svc := service.NewPostService(nil)
	RegisterRoutes(r, "post-test", svc, nilTestSecret, realtime.NewHub(), nil, nilInternalSec)
	return r
}

func makePostToken(t *testing.T, role string) string {
	t.Helper()
	now := time.Now()
	claims := middleware.Claims{
		UserID: "11111111-1111-1111-1111-111111111111",
		Email:  "alice@breezy.dev",
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(nilTestSecret))
	if err != nil {
		t.Fatalf("makePostToken: %v", err)
	}
	return tok
}

// ─── PostStats — ids vide → 200 sans appel repo ───────────────────────────────

func TestPostStats_IdsVides_200(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /posts/stats (ids vide) = %d, attendu 200", w.Code)
	}
}

// ─── CreatePost — gardes de handler avant service ─────────────────────────────

func TestCreatePost_JSONInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(`{invalid}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreatePost(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// Post vide (pas de texte, pas de média, pas de sondage) → 400 avant service.
func TestCreatePost_PostVide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(`{"content":""}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreatePost(vide) = %d, attendu 400", w.Code)
	}
}

// ─── GetPost — ID invalide → ErrInvalidID → 400 ──────────────────────────────

func TestGetPost_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts/not-a-valid-objectid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("GetPost(ID invalide) = %d, attendu 400", w.Code)
	}
}

// ─── UpdatePost — handler garde ───────────────────────────────────────────────

func TestUpdatePost_JSONInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/posts/507f1f77bcf86cd799439011", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("UpdatePost(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// ─── DeletePost — ID invalide → ErrInvalidID → 400 ───────────────────────────

func TestDeletePost_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/not-valid", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("DeletePost(ID invalide) = %d, attendu 400", w.Code)
	}
}

// ─── PinPost / UnpinPost — ID invalide → 400 ─────────────────────────────────

func TestPinPost_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/posts/not-valid/pin", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("PinPost(ID invalide) = %d, attendu 400", w.Code)
	}
}

func TestUnpinPost_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/posts/not-valid/pin", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("UnpinPost(ID invalide) = %d, attendu 400", w.Code)
	}
}

// ─── VotePoll — ID invalide → 400 ────────────────────────────────────────────

func TestVotePoll_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/not-valid/poll/vote", strings.NewReader(`{"choice_id":"c1"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("VotePoll(ID invalide) = %d, attendu 400", w.Code)
	}
}

// ─── ClosePoll — ID invalide → 400 ───────────────────────────────────────────

func TestClosePoll_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/posts/not-valid/poll/close", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("ClosePoll(ID invalide) = %d, attendu 400", w.Code)
	}
}

// ─── ListHidden — route modération : nil repo → 500 avec token mod ───────────

func TestListHidden_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "moderator")
	req := httptest.NewRequest(http.MethodGet, "/posts/moderation/deleted", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListHidden(nil repo, mod) = %d, attendu 500", w.Code)
	}
}

// ─── RestorePost — ID invalide → ErrForbidden ou ErrInvalidID ────────────────

func TestRestorePost_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "moderator")
	req := httptest.NewRequest(http.MethodPost, "/posts/not-valid/restore", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("RestorePost(ID invalide) = %d, attendu 400", w.Code)
	}
}

// ─── PurgePost — ID invalide → 400 ───────────────────────────────────────────

func TestPurgePost_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "moderator")
	req := httptest.NewRequest(http.MethodDelete, "/posts/not-valid/purge", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("PurgePost(ID invalide) = %d, attendu 400", w.Code)
	}
}

// ─── AutoHide / AutoUnhide — InternalHandler ─────────────────────────────────

func TestAutoHide_SansSecret_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/internal/posts/507f1f77bcf86cd799439011/auto-hide", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("AutoHide(sans secret) = %d, attendu 401", w.Code)
	}
}

func TestAutoHide_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/internal/posts/not-valid/auto-hide", nil)
	req.Header.Set("X-Internal-Secret", nilInternalSec)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("AutoHide(ID invalide) = %d, attendu 400", w.Code)
	}
}

func TestAutoUnhide_SansSecret_401(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/internal/posts/507f1f77bcf86cd799439011/auto-unhide", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("AutoUnhide(sans secret) = %d, attendu 401", w.Code)
	}
}

func TestAutoUnhide_IDInvalide_400(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/internal/posts/not-valid/auto-unhide", nil)
	req.Header.Set("X-Internal-Secret", nilInternalSec)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("AutoUnhide(ID invalide) = %d, attendu 400", w.Code)
	}
}

// ─── ListPosts — nil repo → 500 ───────────────────────────────────────────────

func TestListPosts_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListPosts(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── parseQueryTime — couvert indirectement via ListHidden ───────────────────
// ListHidden avec since/until invalides → parseQueryTime retourne nil → continue.
func TestListHidden_SinceInvalide_NilRepo_500(t *testing.T) {
	r := newNilRepoRouter(t)
	tok := makePostToken(t, "moderator")
	req := httptest.NewRequest(http.MethodGet, "/posts/moderation/deleted?since=not-a-date", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// parseQueryTime retourne nil → service est appelé → nil repo → 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListHidden(since invalide, nil repo) = %d, attendu 500", w.Code)
	}
}
