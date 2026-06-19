package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── GetByUserID — route publique, nil repo → panic → 500 ────────────────────

func TestGetByUserID_NilRepo_500(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/profils/some-user-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("GET /profils/:userId (nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── GetActivity — route publique, nil repo → panic → 500 ───────────────────

func TestGetActivity_NilRepo_500(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/profils/some-user-id/activity", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("GET /profils/:userId/activity (nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── GetMe — token valide, nil repo → panic → 500 ────────────────────────────

func TestGetMe_AvecToken_NilRepo_500(t *testing.T) {
	r := newHandlerRouter(t)
	tok := makeProfilToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/profils/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("GET /profils/me (nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── TouchActivity — token valide, nil repo → panic → 500 ───────────────────

func TestTouchActivity_AvecToken_NilRepo_500(t *testing.T) {
	r := newHandlerRouter(t)
	tok := makeProfilToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/profils/me/activity", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("PATCH /profils/me/activity (nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── TouchActivityOffline — token valide, nil repo → panic → 500 ─────────────

func TestTouchActivityOffline_AvecToken_NilRepo_500(t *testing.T) {
	r := newHandlerRouter(t)
	tok := makeProfilToken(t, "user")
	req := httptest.NewRequest(http.MethodPatch, "/profils/me/activity/offline", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("PATCH /profils/me/activity/offline (nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── Create — display_name invalide → ErrInvalidDisplayName → respondProfilError → 400

func TestCreate_DisplayNameInvalide_400(t *testing.T) {
	r := newHandlerRouter(t)
	tok := makeProfilToken(t, "user")
	// display_name avec caractère interdit (€) → ErrInvalidDisplayName avant repo
	body := `{"display_name":"Alice€"}`
	req := httptest.NewRequest(http.MethodPost, "/profils", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Create(display_name invalide) = %d, attendu 400", w.Code)
	}
}

// display_name vide → ErrInvalidDisplayName → 400.
func TestCreate_DisplayNameVide_400(t *testing.T) {
	r := newHandlerRouter(t)
	tok := makeProfilToken(t, "user")
	body := `{"display_name":""}`
	req := httptest.NewRequest(http.MethodPost, "/profils", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Create(display_name vide) = %d, attendu 400", w.Code)
	}
}

// ─── respondProfilError — default branch (via nil repo panic → 500) ───────────

// La branche default de respondProfilError est couverte par les panics ci-dessus
// qui passent par gin.Recovery() avant d'atteindre le handler.
// On couvre ici le chemin explicite : Create avec display_name valide → nil repo
// → panic → 500 → recovery → respondProfilError n'est pas appelée (panic bypass),
// mais les tests Create_DisplayNameInvalide_400 couvrent bien respondProfilError.

func TestDelete_SansToken_401(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/profils/some-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("DELETE /profils/:id sans token = %d, attendu 401", w.Code)
	}
}

// ─── AdminCreate ──────────────────────────────────────────────────────────────

func TestAdminCreate_JSONInvalide_400(t *testing.T) {
	r := newHandlerRouter(t)
	tok := makeProfilToken(t, "admin")
	req := httptest.NewRequest(http.MethodPost, "/profils/admin", strings.NewReader(`{invalid}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("AdminCreate JSON invalide = %d, attendu 400", w.Code)
	}
}

// ─── GetVisibility / GetLikesVisibility — nil repo → 500 ─────────────────────

func TestGetVisibility_NilRepo_500(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/profils/some-id/visibility", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("GET /profils/:userId/visibility (nil repo) = %d, attendu 500", w.Code)
	}
}

func TestGetLikesVisibility_NilRepo_500(t *testing.T) {
	r := newHandlerRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/profils/some-id/likes-visibility", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("GET /profils/:userId/likes-visibility (nil repo) = %d, attendu 500", w.Code)
	}
}
