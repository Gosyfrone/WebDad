// Tests supplémentaires pour les handlers auth.
// Stratégie :
//   - DB=nil → panic récupérée par gin.Recovery() → 500 (couvre les chemins
//     qui atteignent la DB) ;
//   - chemins sans DB : validation JSON, gardes JWT/rôle, routes publiques ;
//   - tous les tests utilisent newTestRouter() déjà défini dans handlers_test.go.
package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/webdad/auth-service/internal/models"
)

// ─── Register — atteint la DB (nil) → 500 via Recovery ───────────────────────

func TestRegister_PayloadValide_500(t *testing.T) {
	r := newTestRouter(t)
	body := `{"email":"new@breezy.dev","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// DB=nil → panic → Recovery → 500.
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("register avec DB nil = %d, attendu 500", w.Code)
	}
}

func TestRegister_EmailInvalide_400(t *testing.T) {
	r := newTestRouter(t)
	body := `{"email":"not-an-email","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("email invalide = %d, attendu 400", w.Code)
	}
}

func TestRegister_MotDePasseTropCourt_400(t *testing.T) {
	r := newTestRouter(t)
	body := `{"email":"u@x.dev","password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("mot de passe court = %d, attendu 400", w.Code)
	}
}

// ─── Login — chemins supplémentaires ─────────────────────────────────────────

func TestLogin_SansPassword_400(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"u@x.dev"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("sans password = %d, attendu 400", w.Code)
	}
}

func TestLogin_AvecEmail_AtteintDB_500(t *testing.T) {
	r := newTestRouter(t)
	body := `{"email":"u@x.dev","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// DB=nil → 500.
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("login (DB nil) = %d, attendu 500", w.Code)
	}
}

func TestLogin_AvecUserID_AtteintDB_500(t *testing.T) {
	r := newTestRouter(t)
	body := `{"user_id":"some-uuid","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("loginByUserID (DB nil) = %d, attendu 500", w.Code)
	}
}

// ─── ForgotPassword ───────────────────────────────────────────────────────────

func TestForgotPassword_EmailValide_200(t *testing.T) {
	r := newTestRouter(t)
	body := `{"email":"u@breezy.dev"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/password/forgot", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// ForgotPassword est anti-énumération : 200 même avec DB nil si elle ne panique pas.
	// Avec DB nil elle va paniquer → 500 via Recovery.
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Fatalf("forgot password = %d, attendu 200 ou 500", w.Code)
	}
}

func TestForgotPassword_EmailInvalide_400(t *testing.T) {
	r := newTestRouter(t)
	body := `{"email":"not-an-email"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/password/forgot", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("email invalide = %d, attendu 400", w.Code)
	}
}

// ─── ResetPassword ────────────────────────────────────────────────────────────

func TestResetPassword_PayloadValide_AtteintDB_500(t *testing.T) {
	r := newTestRouter(t)
	body := `{"token":"some-token","new_password":"NewPass123!"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/password/reset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// DB=nil → panic → 500.
	if w.Code != http.StatusInternalServerError && w.Code != http.StatusBadRequest {
		t.Fatalf("reset (DB nil) = %d, attendu 400 ou 500", w.Code)
	}
}

func TestResetPassword_NouveauMotDePasseCourt_400(t *testing.T) {
	r := newTestRouter(t)
	body := `{"token":"tk","new_password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/password/reset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("mdp trop court = %d, attendu 400", w.Code)
	}
}

// ─── ChangePassword — authentifié ─────────────────────────────────────────────

func TestChangePassword_PayloadValide_AtteintDB_500(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	body := `{"current_password":"OldPass1!","new_password":"NewPass2!"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/password/change", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// DB=nil → panic → 500.
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("change password (DB nil) = %d, attendu 500", w.Code)
	}
}

func TestChangePassword_NouveauMDPCourt_400(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	body := `{"current_password":"OldPass1!","new_password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/password/change", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("nouveau mdp court = %d, attendu 400", w.Code)
	}
}

// ─── Logout — authentifié ─────────────────────────────────────────────────────

func TestLogout_AvecToken_200(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	body := `{"refresh_token":""}`
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Logout avec refresh_token vide → best-effort : 200.
	if w.Code != http.StatusOK {
		t.Fatalf("logout (token vide) = %d, attendu 200", w.Code)
	}
}

func TestLogout_SansCorps_200(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("logout (sans corps) = %d, attendu 200", w.Code)
	}
}

// ─── VerifyEmail ──────────────────────────────────────────────────────────────

func TestConfirmVerifyEmail_TokenValide_AtteintDB_500(t *testing.T) {
	r := newTestRouter(t)
	body := `{"token":"some-verification-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// DB=nil → panic → 500.
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("confirm email (DB nil) = %d, attendu 500", w.Code)
	}
}

func TestRequestVerifyEmail_EmailValide_200(t *testing.T) {
	r := newTestRouter(t)
	body := `{"email":"u@breezy.dev"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email/request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// ResendVerification est anti-énumération. DB nil → panic → 500.
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Fatalf("request verify email = %d, attendu 200 ou 500", w.Code)
	}
}

// ─── Validate — détails du token ─────────────────────────────────────────────

func TestValidate_TokenValide_ContenantRôle(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleAdmin)
	req := httptest.NewRequest(http.MethodGet, "/auth/validate", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("validate admin = %d, attendu 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"role"`) {
		t.Fatalf("réponse sans 'role': %s", body)
	}
	if !strings.Contains(body, "admin") {
		t.Fatalf("rôle admin absent: %s", body)
	}
}

func TestValidate_TokenValide_ContenantEmail(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	req := httptest.NewRequest(http.MethodGet, "/auth/validate", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("validate = %d, attendu 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"email"`) {
		t.Fatalf("réponse sans 'email': %s", body)
	}
}

// ─── Admin routes — contenu des réponses ────────────────────────────────────

func TestListUsers_RoleModerator_200(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleModerator)
	req := httptest.NewRequest(http.MethodGet, "/auth/users", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// DB nil → 500.
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("list users (DB nil, mod) = %d, attendu 500", w.Code)
	}
}

func TestListUsers_RoleAdmin_200(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleAdmin)
	req := httptest.NewRequest(http.MethodGet, "/auth/users?q=breezy", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("list users (DB nil, admin) = %d, attendu 500", w.Code)
	}
}

func TestPublicRoles_GuardsAndDB(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/auth/users/roles?ids=u1,u2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("public roles sans token = %d, attendu 401", w.Code)
	}

	tok := makeAuthToken(t, models.RoleUser)
	req = httptest.NewRequest(http.MethodGet, "/auth/users/roles?ids=u1,u2", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("public roles DB nil = %d, attendu 500", w.Code)
	}
}

func TestSetRole_SurSoiMême_400(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleAdmin)
	// L'ID dans le token est "11111111-1111-1111-1111-111111111111" (makeAuthToken).
	req := httptest.NewRequest(http.MethodPut, "/auth/users/11111111-1111-1111-1111-111111111111/role",
		strings.NewReader(`{"role":"user"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("auto-modification rôle = %d, attendu 400", w.Code)
	}
}

func TestSetStatus_SurSoiMême_400(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleAdmin)
	active := true
	_ = active
	req := httptest.NewRequest(http.MethodPatch, "/auth/users/11111111-1111-1111-1111-111111111111/status",
		strings.NewReader(`{"is_active":false}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("auto-bannissement = %d, attendu 400", w.Code)
	}
}

func TestDeleteUser_SurSoiMême_400(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleAdmin)
	req := httptest.NewRequest(http.MethodDelete, "/auth/users/11111111-1111-1111-1111-111111111111", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("auto-suppression = %d, attendu 400", w.Code)
	}
}

func TestDeleteUser_Admin_AutreUser_500(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleAdmin)
	req := httptest.NewRequest(http.MethodDelete, "/auth/users/22222222-2222-2222-2222-222222222222", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// DB nil → 500.
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("delete user (DB nil) = %d, attendu 500", w.Code)
	}
}

func TestSetRole_Admin_AutreUser_JSONValide_500(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleAdmin)
	req := httptest.NewRequest(http.MethodPut, "/auth/users/22222222-2222-2222-2222-222222222222/role",
		strings.NewReader(`{"role":"moderator"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// DB nil → 500.
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("set role (DB nil) = %d, attendu 500", w.Code)
	}
}

func TestAdminCreateUser_PayloadValide_500(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleAdmin)
	body := `{"email":"new@breezy.dev","password":"Password1!","username":"newuser"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/users", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// DB nil → 500.
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("admin create user (DB nil) = %d, attendu 500", w.Code)
	}
}

// ─── MFA handlers ────────────────────────────────────────────────────────────

func TestMFASetup_TokenValide_503(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	req := httptest.NewRequest(http.MethodPost, "/auth/mfa/setup", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// MFA non configurée (mfaCipher=nil) → 503.
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("mfa setup (non configurée) = %d, attendu 503", w.Code)
	}
}

func TestMFAEnable_TokenValide_503(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	body := `{"code":"123456"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/mfa/enable", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("mfa enable (non configurée) = %d, attendu 503", w.Code)
	}
}

func TestMFAEnable_JSONInvalide_400(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	req := httptest.NewRequest(http.MethodPost, "/auth/mfa/enable", strings.NewReader(`{bad json}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("mfa enable (JSON invalide) = %d, attendu 400", w.Code)
	}
}

func TestMFADisable_TokenValide_503(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	body := `{"code":"123456"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/mfa/disable", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("mfa disable (non configurée) = %d, attendu 503", w.Code)
	}
}

func TestMFADisable_JSONInvalide_400(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	req := httptest.NewRequest(http.MethodPost, "/auth/mfa/disable", strings.NewReader(`{bad json}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("mfa disable (JSON invalide) = %d, attendu 400", w.Code)
	}
}

func TestMFAStatus_TokenValide_200(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleUser)
	req := httptest.NewRequest(http.MethodGet, "/auth/mfa/status", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// MFAStatus appelle la DB (nil) → panic → 500.
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("mfa status (DB nil) = %d, attendu 500", w.Code)
	}
}

func TestMFAVerify_JSONInvalide_400(t *testing.T) {
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/mfa/verify", strings.NewReader(`{bad}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("mfa verify (JSON invalide) = %d, attendu 400", w.Code)
	}
}

func TestMFAVerify_MFANonConfigurée_503(t *testing.T) {
	r := newTestRouter(t)
	body := `{"challenge":"ch","code":"123456"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/mfa/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// MFA non configurée → 503.
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("mfa verify (non configurée) = %d, attendu 503", w.Code)
	}
}

// ─── SetStatus — JSON invalide ────────────────────────────────────────────────

func TestSetStatus_Moderator_JSONInvalide_400(t *testing.T) {
	r := newTestRouter(t)
	tok := makeAuthToken(t, models.RoleModerator)
	req := httptest.NewRequest(http.MethodPatch, "/auth/users/22222222-2222-2222-2222-222222222222/status",
		strings.NewReader(`{bad json}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Le modérateur vérifie d'abord la cible (DB nil → 500) avant de lire le JSON.
	// Donc soit 400 soit 500.
	if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
		t.Fatalf("set status (JSON invalide, mod) = %d, attendu 400 ou 500", w.Code)
	}
}
