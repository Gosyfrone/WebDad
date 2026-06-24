package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/webdad/auth-service/internal/middleware"
	"github.com/webdad/auth-service/internal/models"
	"github.com/webdad/auth-service/internal/services"
)

const handlerSecret = "auth-handler-test-secret"

func newAuthService(t *testing.T) *services.AuthService {
	t.Helper()
	svc, err := services.New(nil, handlerSecret, 15*time.Minute, 24*time.Hour, nil, "", "", false, "")
	if err != nil {
		t.Fatalf("services.New: %v", err)
	}
	return svc
}

// newTestRouter construit un routeur Gin avec les handlers auth et gin.Recovery().
// Les appels qui atteignent la DB (nil) paniqueront → Recovery → 500.
func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	auth := newAuthService(t)
	h := New(auth, nil)
	jwtMW := middleware.JWTAuth(auth)

	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)
	r.POST("/auth/password/forgot", h.ForgotPassword)
	r.POST("/auth/password/reset", h.ResetPassword)
	r.POST("/auth/password/change", jwtMW, h.ChangePassword)
	r.POST("/auth/verify-email/confirm", h.ConfirmVerifyEmail)
	r.POST("/auth/verify-email/request", h.RequestVerifyEmail)
	r.POST("/auth/mfa/setup", jwtMW, h.MFASetup)
	r.POST("/auth/mfa/enable", jwtMW, h.MFAEnable)
	r.POST("/auth/mfa/disable", jwtMW, h.MFADisable)
	r.GET("/auth/mfa/status", jwtMW, h.MFAStatus)
	r.POST("/auth/mfa/verify", h.MFAVerify)
	r.POST("/auth/logout", jwtMW, h.Logout)
	r.GET("/auth/validate", jwtMW, h.Validate)
	r.POST("/auth/users", jwtMW, middleware.AdminOnly(), h.AdminCreateUser)
	r.PUT("/auth/users/:id/role", jwtMW, middleware.AdminOnly(), h.SetRole)
	r.PATCH("/auth/users/:id/status", jwtMW, middleware.ModeratorOnly(), h.SetStatus)
	r.DELETE("/auth/users/:id", jwtMW, middleware.AdminOnly(), h.DeleteUser)
	r.GET("/auth/users/roles", jwtMW, h.PublicRoles)
	r.GET("/auth/users", jwtMW, middleware.ModeratorOnly(), h.ListUsers)
	return r
}

func makeAuthToken(t *testing.T, role string) string {
	t.Helper()
	now := time.Now()
	claims := &services.Claims{
		UserID: "11111111-1111-1111-1111-111111111111",
		Email:  "user@breezy.dev",
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(handlerSecret))
	if err != nil {
		t.Fatalf("makeAuthToken: %v", err)
	}
	return tok
}

// ─── Register ─────────────────────────────────────────────────────────────────

func TestRegister_JSONInvalide_400(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("JSON invalide = %d, attendu 400", w.Code)
	}
}

func TestRegister_PayloadManquant_400(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("payload vide = %d, attendu 400", w.Code)
	}
}

// ─── Login ────────────────────────────────────────────────────────────────────

func TestLogin_JSONInvalide_400(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{bad json}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("JSON invalide = %d, attendu 400", w.Code)
	}
}

func TestLogin_SansEmailNiUserID_400(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"password":"p"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("sans email/user_id = %d, attendu 400", w.Code)
	}
}

// ─── ForgotPassword ───────────────────────────────────────────────────────────

func TestForgotPassword_JSONInvalide_400(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/password/forgot", strings.NewReader(`{bad}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("JSON invalide = %d, attendu 400", w.Code)
	}
}

// ─── ResetPassword ────────────────────────────────────────────────────────────

func TestResetPassword_JSONInvalide_400(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/password/reset", strings.NewReader(`{bad}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("JSON invalide = %d, attendu 400", w.Code)
	}
}

// ─── ChangePassword — JWT requis ──────────────────────────────────────────────

func TestChangePassword_SansToken_401(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/password/change", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sans token = %d, attendu 401", w.Code)
	}
}

func TestChangePassword_JSONInvalide_400(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	req := httptest.NewRequest(http.MethodPost, "/auth/password/change", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("JSON invalide = %d, attendu 400", w.Code)
	}
}

// ─── Validate ────────────────────────────────────────────────────────────────

func TestValidate_SansToken_401(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/auth/validate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sans token = %d, attendu 401", w.Code)
	}
}

func TestValidate_TokenValide_200(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	req := httptest.NewRequest(http.MethodGet, "/auth/validate", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("token valide = %d, attendu 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"user_id"`) {
		t.Fatalf("réponse sans 'user_id' : %s", w.Body.String())
	}
}

// ─── Admin routes — gardes d'accès ───────────────────────────────────────────

func TestAdminCreateUser_SansToken_401(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/users", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sans token = %d, attendu 401", w.Code)
	}
}

func TestAdminCreateUser_RoleUser_403(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	req := httptest.NewRequest(http.MethodPost, "/auth/users", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("role user = %d, attendu 403", w.Code)
	}
}

func TestAdminCreateUser_JSONInvalide_400(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleAdmin)
	req := httptest.NewRequest(http.MethodPost, "/auth/users", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("JSON invalide = %d, attendu 400", w.Code)
	}
}

func TestSetRole_RoleUser_403(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	req := httptest.NewRequest(http.MethodPut, "/auth/users/some-id/role", strings.NewReader(`{"role":"user"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("role user = %d, attendu 403", w.Code)
	}
}

func TestSetRole_JSONInvalide_400(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleAdmin)
	req := httptest.NewRequest(http.MethodPut, "/auth/users/some-id/role", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("JSON invalide = %d, attendu 400", w.Code)
	}
}

func TestSetStatus_SansToken_401(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPatch, "/auth/users/some-id/status", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sans token = %d, attendu 401", w.Code)
	}
}

func TestSetStatus_RoleUser_403(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	req := httptest.NewRequest(http.MethodPatch, "/auth/users/some-id/status", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("role user = %d, attendu 403", w.Code)
	}
}

func TestDeleteUser_RoleUser_403(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	req := httptest.NewRequest(http.MethodDelete, "/auth/users/some-id", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("role user = %d, attendu 403", w.Code)
	}
}

func TestListUsers_SansToken_401(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/auth/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sans token = %d, attendu 401", w.Code)
	}
}

// ─── MFA — JWT requis ─────────────────────────────────────────────────────────

func TestMFASetup_SansToken_401(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/mfa/setup", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sans token = %d, attendu 401", w.Code)
	}
}

func TestMFAStatus_SansToken_401(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/auth/mfa/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sans token = %d, attendu 401", w.Code)
	}
}

// ─── Logout — JWT requis ──────────────────────────────────────────────────────

func TestLogout_SansToken_401(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sans token = %d, attendu 401", w.Code)
	}
}

// ─── VerifyEmail ─────────────────────────────────────────────────────────────

func TestConfirmVerifyEmail_JSONInvalide_400(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email/confirm", strings.NewReader(`{bad}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("JSON invalide = %d, attendu 400", w.Code)
	}
}

func TestRequestVerifyEmail_JSONInvalide_400(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email/request", strings.NewReader(`{bad}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("JSON invalide = %d, attendu 400", w.Code)
	}
}
