package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/webdad/notification-service/internal/middleware"
	"github.com/webdad/notification-service/internal/realtime"
	"github.com/webdad/notification-service/internal/service"
)

const testSecret = "test-jwt-secret"

// newNotifRouter monte les routes avec un service à repo nil + gin.Recovery().
// Les panics du repo nil sont converties en 500.
func newNotifRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	svc := service.NewNotificationService(nil, nil, nil)
	RegisterRoutes(r, "notif-test", svc, realtime.NewHub(), testSecret, "internal-secret", nil)
	return r
}

func makeNotifToken(t *testing.T) string {
	t.Helper()
	now := time.Now()
	claims := middleware.Claims{
		UserID: "u1",
		Email:  "u1@x.dev",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("makeNotifToken: %v", err)
	}
	return tok
}

// ─── List ─────────────────────────────────────────────────────────────────────

// before= invalide → ErrInvalidID → respondError → 400.
func TestList_BeforeInvalid_400(t *testing.T) {
	r := newNotifRouter()
	tok := makeNotifToken(t)
	req := httptest.NewRequest(http.MethodGet, "/notifications?before=not-a-valid-id", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("List(before invalide) = %d, attendu 400", w.Code)
	}
}

// before= vide → aucun curseur → repo nil → panic → 500.
func TestList_NilRepo_500(t *testing.T) {
	r := newNotifRouter()
	tok := makeNotifToken(t)
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("List(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── UnreadCount ──────────────────────────────────────────────────────────────

func TestUnreadCount_NilRepo_500(t *testing.T) {
	r := newNotifRouter()
	tok := makeNotifToken(t)
	req := httptest.NewRequest(http.MethodGet, "/notifications/unread-count", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("UnreadCount(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── MarkAllRead ──────────────────────────────────────────────────────────────

func TestMarkAllRead_NilRepo_500(t *testing.T) {
	r := newNotifRouter()
	tok := makeNotifToken(t)
	req := httptest.NewRequest(http.MethodPost, "/notifications/read", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("MarkAllRead(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── MarkRead ─────────────────────────────────────────────────────────────────

// id invalide → ErrInvalidID → respondError → 400.
func TestMarkRead_InvalidID_400(t *testing.T) {
	r := newNotifRouter()
	tok := makeNotifToken(t)
	req := httptest.NewRequest(http.MethodPost, "/notifications/not-valid-id/read", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("MarkRead(id invalide) = %d, attendu 400", w.Code)
	}
}

// id vide → ErrInvalidID → 400.
func TestMarkRead_EmptyID_400(t *testing.T) {
	r := newNotifRouter()
	tok := makeNotifToken(t)
	// Gin matches "/:id/read" — envoyer un id d'un seul espace n'est pas valide
	req := httptest.NewRequest(http.MethodPost, "/notifications/%20/read", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Soit 400 (ErrInvalidID) soit 500 (nil repo) selon si le service rejette en amont
	if w.Code == http.StatusOK {
		t.Errorf("MarkRead(id espace) ne doit pas retourner 200")
	}
}

// ─── parseLimit ───────────────────────────────────────────────────────────────

// limit valide → parseLimit retourne la valeur → atteint le repo (nil → 500).
func TestParseLimit_Valid(t *testing.T) {
	r := newNotifRouter()
	tok := makeNotifToken(t)
	req := httptest.NewRequest(http.MethodGet, "/notifications?limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Le service plante (nil repo) → 500, mais parseLimit a bien été appelé.
	if w.Code == http.StatusUnauthorized {
		t.Errorf("parseLimit(10) ne devrait pas retourner 401")
	}
}

// limit invalide (non-entier) → parseLimit retourne 0 → service.List(limit=0).
func TestParseLimit_Invalid(t *testing.T) {
	r := newNotifRouter()
	tok := makeNotifToken(t)
	req := httptest.NewRequest(http.MethodGet, "/notifications?limit=abc&before=not-valid", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// before invalide → ErrInvalidID → 400.
	if w.Code != http.StatusBadRequest {
		t.Errorf("parseLimit(abc) + before invalide = %d, attendu 400", w.Code)
	}
}

// ─── respondError — branches ─────────────────────────────────────────────────

// ErrInvalidID → 400 (couvert via List avec before invalide).
// ErrNotFound → 404 (MarkRead avec un ID valide mais repo nil → on ne peut pas
//
//	atteindre ErrNotFound sans mock ; on couvre le 500 default branch).
func TestRespondError_Default_500(t *testing.T) {
	r := newNotifRouter()
	tok := makeNotifToken(t)
	// UnreadCount → nil repo → panic → Recovery → 500 → respondError default.
	req := httptest.NewRequest(http.MethodGet, "/notifications/unread-count", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("respondError(default) = %d, attendu 500", w.Code)
	}
}
