package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/webdad/user-service/internal/middleware"
	"github.com/webdad/user-service/internal/service"
)

// newFullRouter enregistre toutes les routes (y compris follows) avec repo nil.
// gin.Recovery() transforme les panics (nil repo) en 500.
func newFullRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	svc := service.New(nil, 0)
	h := New(svc)
	auth := middleware.JWTAuth(handlerTestSecret)

	// users
	r.GET("/users", h.List)
	r.GET("/users/search", h.Search)
	r.GET("/users/suggestions", h.Suggestions)
	r.GET("/users/:id", h.GetByID)
	r.GET("/users/by-username/:username", h.GetByUsername)
	r.POST("/users", auth, h.Create)
	r.POST("/users/admin", auth, middleware.AdminOnly(), h.AdminCreate)
	r.DELETE("/users/:id", auth, middleware.AdminOnly(), h.Delete)
	r.DELETE("/users/:id/purge", auth, middleware.AdminOnly(), h.PurgeUser)
	r.GET("/users/me", auth, h.GetMe)
	r.PATCH("/users/me", auth, h.UpdateMe)
	r.PATCH("/users/:id/status", auth, middleware.ModeratorOnly(), h.SetStatus)

	// follows
	r.POST("/users/:id/follow", auth, h.Follow)
	r.DELETE("/users/:id/follow", auth, h.Unfollow)
	r.DELETE("/users/me/followers/:id", auth, h.RemoveFollower)
	r.GET("/users/:id/followers", h.Followers)
	r.GET("/users/:id/following", h.Following)
	r.POST("/users/follow-requests/:followerId/accept", auth, h.AcceptFollowRequest)
	r.POST("/users/follow-requests/:followerId/reject", auth, h.RejectFollowRequest)
	r.GET("/users/me/follow-requests/outgoing", auth, h.PendingFollowRequests)
	r.POST("/internal/users/:ownerId/accept-all-follow-requests", h.AcceptAllFollowRequests)
	return r
}

// ─── List / Suggestions / GetByID / GetByUsername ────────────────────────────

func TestList_NilRepo_500(t *testing.T) {
	r := newFullRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("GET /users (nil repo) = %d, attendu 500", w.Code)
	}
}

func TestSuggestions_NilRepo_500(t *testing.T) {
	r := newFullRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/users/suggestions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("GET /users/suggestions (nil repo) = %d, attendu 500", w.Code)
	}
}

func TestGetByID_NilRepo_500(t *testing.T) {
	r := newFullRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/users/some-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("GET /users/:id (nil repo) = %d, attendu 500", w.Code)
	}
}

func TestGetByUsername_NilRepo_500(t *testing.T) {
	r := newFullRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/users/by-username/alice", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("GET /users/by-username/:username (nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── Delete / PurgeUser ──────────────────────────────────────────────────────

func TestDelete_SansToken_401(t *testing.T) {
	r := newFullRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/users/some-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("DELETE /users/:id sans token = %d, attendu 401", w.Code)
	}
}

func TestDelete_RoleUser_403(t *testing.T) {
	r := newFullRouter(t)
	tok := makeHandlerToken(t, "user")
	req := httptest.NewRequest(http.MethodDelete, "/users/some-id", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("DELETE /users/:id (user) = %d, attendu 403", w.Code)
	}
}

func TestPurgeUser_SansToken_401(t *testing.T) {
	r := newFullRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/users/some-id/purge", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("DELETE /users/:id/purge sans token = %d, attendu 401", w.Code)
	}
}

// ─── Follow ──────────────────────────────────────────────────────────────────

func TestFollow_SansToken_401(t *testing.T) {
	r := newFullRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/users/some-id/follow", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("POST /users/:id/follow sans token = %d, attendu 401", w.Code)
	}
}

// Follow(self) → ErrSelfFollow → respondUserError → 400.
// Le claims.UserID de makeHandlerToken est "11111111-…-111111111111".
func TestFollow_Self_400(t *testing.T) {
	r := newFullRouter(t)
	tok := makeHandlerToken(t, "user")
	// L'ID dans le path est le même que claims.UserID
	req := httptest.NewRequest(http.MethodPost, "/users/11111111-1111-1111-1111-111111111111/follow", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Follow(self) = %d, attendu 400", w.Code)
	}
}

// ─── Unfollow ─────────────────────────────────────────────────────────────────

func TestUnfollow_SansToken_401(t *testing.T) {
	r := newFullRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/users/some-id/follow", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("DELETE /users/:id/follow sans token = %d, attendu 401", w.Code)
	}
}

// ─── RemoveFollower ───────────────────────────────────────────────────────────

func TestRemoveFollower_SansToken_401(t *testing.T) {
	r := newFullRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/users/me/followers/u2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("DELETE /users/me/followers/:id sans token = %d, attendu 401", w.Code)
	}
}

func TestRemoveFollower_Self_400(t *testing.T) {
	r := newFullRouter(t)
	tok := makeHandlerToken(t, "user")
	// Retire soi-même → ErrSelfFollow
	req := httptest.NewRequest(http.MethodDelete, "/users/me/followers/11111111-1111-1111-1111-111111111111", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("RemoveFollower(self) = %d, attendu 400", w.Code)
	}
}

// ─── Followers / Following / IsFollowing ─────────────────────────────────────

func TestFollowers_NilRepo_500(t *testing.T) {
	r := newFullRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/users/u1/followers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("GET /users/:id/followers (nil repo) = %d, attendu 500", w.Code)
	}
}

func TestFollowing_NilRepo_500(t *testing.T) {
	r := newFullRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/users/u1/following", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("GET /users/:id/following (nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── AcceptFollowRequest / RejectFollowRequest ────────────────────────────────

func TestAcceptFollowRequest_SansToken_401(t *testing.T) {
	r := newFullRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/users/follow-requests/u2/accept", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("AcceptFollowRequest sans token = %d, attendu 401", w.Code)
	}
}

func TestRejectFollowRequest_SansToken_401(t *testing.T) {
	r := newFullRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/users/follow-requests/u2/reject", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("RejectFollowRequest sans token = %d, attendu 401", w.Code)
	}
}

// ─── PendingFollowRequests ───────────────────────────────────────────────────

func TestPendingFollowRequests_SansToken_401(t *testing.T) {
	r := newFullRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/users/me/follow-requests/outgoing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("PendingFollowRequests sans token = %d, attendu 401", w.Code)
	}
}

// ─── AcceptAllFollowRequests — interne ────────────────────────────────────────

func TestAcceptAllFollowRequests_NilRepo_500(t *testing.T) {
	r := newFullRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/internal/users/owner1/accept-all-follow-requests", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("AcceptAllFollowRequests (nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── respondUserError — couverture des branches via handlers ─────────────────

// ErrInvalidUsername → 400 via Create.
func TestRespondUserError_InvalidUsername_400(t *testing.T) {
	r := newFullRouter(t)
	tok := makeHandlerToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"username":"ab"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("ErrInvalidUsername → %d, attendu 400", w.Code)
	}
}

// ErrSelfFollow → 400 via Follow.
func TestRespondUserError_SelfFollow_400(t *testing.T) {
	r := newFullRouter(t)
	tok := makeHandlerToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/users/11111111-1111-1111-1111-111111111111/follow", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("ErrSelfFollow → %d, attendu 400", w.Code)
	}
}

// ErrInvalidLocale → 400 via UpdateMe.
func TestRespondUserError_InvalidLocale_400(t *testing.T) {
	r := newFullRouter(t)
	tok := makeHandlerToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/users/me",
		strings.NewReader(`{"preferred_locale":"xx"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("ErrInvalidLocale → %d, attendu 400", w.Code)
	}
}
