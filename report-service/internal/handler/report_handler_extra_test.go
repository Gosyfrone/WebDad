package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/webdad/report-service/internal/service"
)

// newRecoveryRouter est identique à newTestRouter mais ajoute gin.Recovery()
// afin que les panics (nil repo dans le service) soient converties en 500.
func newRecoveryRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	RegisterRoutes(r, "report-service", (*service.ReportService)(nil), testSecret)
	return r
}

func makeReportToken(t *testing.T, role string) string {
	t.Helper()
	return makeToken(t, testSecret, role, time.Hour)
}

// ─── Create — gardes handler/service avant repo ───────────────────────────────

// category requis → binding fails → 400.
func TestCreate_CategoryManquante_400(t *testing.T) {
	r := newRecoveryRouter(t)
	tok := makeReportToken(t, "user")
	body := `{"reason":"spam","entity_type":"post"}`
	req := httptest.NewRequest(http.MethodPost, "/reports", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Create(category manquante) = %d, attendu 400", w.Code)
	}
}

// reason requis → binding fails → 400.
func TestCreate_ReasonManquant_400(t *testing.T) {
	r := newRecoveryRouter(t)
	tok := makeReportToken(t, "user")
	body := `{"category":"moderation"}`
	req := httptest.NewRequest(http.MethodPost, "/reports", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Create(reason manquant) = %d, attendu 400", w.Code)
	}
}

// JSON invalide → binding fails → 400.
func TestCreate_JSONInvalide_400(t *testing.T) {
	r := newRecoveryRouter(t)
	tok := makeReportToken(t, "user")
	req := httptest.NewRequest(http.MethodPost, "/reports", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Create(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// reason invalide → service.ErrValidation → respondError → 400.
func TestCreate_ReasonInvalide_400(t *testing.T) {
	r := newRecoveryRouter(t)
	tok := makeReportToken(t, "user")
	body := `{"category":"moderation","reason":"not-a-valid-reason"}`
	req := httptest.NewRequest(http.MethodPost, "/reports", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Create(reason invalide) = %d, attendu 400", w.Code)
	}
}

// ─── Reply — JSON binding guard ───────────────────────────────────────────────

func TestReply_JSONInvalide_400(t *testing.T) {
	r := newRecoveryRouter(t)
	tok := makeReportToken(t, "moderator")
	req := httptest.NewRequest(http.MethodPost, "/reports/tickets/some-id/replies", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Reply(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// text requis → binding fails → 400.
func TestReply_TextManquant_400(t *testing.T) {
	r := newRecoveryRouter(t)
	tok := makeReportToken(t, "moderator")
	req := httptest.NewRequest(http.MethodPost, "/reports/tickets/some-id/replies", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Reply(text manquant) = %d, attendu 400", w.Code)
	}
}

// ─── ChangeStatus — JSON binding guard ────────────────────────────────────────

func TestChangeStatus_JSONInvalide_400(t *testing.T) {
	r := newRecoveryRouter(t)
	tok := makeReportToken(t, "moderator")
	req := httptest.NewRequest(http.MethodPatch, "/reports/tickets/some-id/status", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("ChangeStatus(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// ─── UpdateSettings — JSON binding guard ─────────────────────────────────────

func TestUpdateSettings_JSONInvalide_400(t *testing.T) {
	r := newRecoveryRouter(t)
	tok := makeReportToken(t, "admin")
	req := httptest.NewRequest(http.MethodPatch, "/reports/settings", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("UpdateSettings(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// ─── IssueWarning — JSON binding guard ───────────────────────────────────────

func TestIssueWarning_JSONInvalide_400(t *testing.T) {
	r := newRecoveryRouter(t)
	tok := makeReportToken(t, "moderator")
	req := httptest.NewRequest(http.MethodPost, "/reports/warnings", strings.NewReader(`{bad}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("IssueWarning(JSON invalide) = %d, attendu 400", w.Code)
	}
}

// ─── GetTicket — ID invalide → ErrInvalidID → 400 ────────────────────────────

func TestGetTicket_IDInvalide_400(t *testing.T) {
	r := newRecoveryRouter(t)
	tok := makeReportToken(t, "moderator")
	req := httptest.NewRequest(http.MethodGet, "/reports/tickets/not-valid", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("GetTicket(ID invalide) = %d, attendu 400", w.Code)
	}
}

// ─── ListTickets — status invalide → ErrValidation → 400 ────────────────────

func TestListTickets_StatusInvalide_400(t *testing.T) {
	r := newRecoveryRouter(t)
	tok := makeReportToken(t, "moderator")
	req := httptest.NewRequest(http.MethodGet, "/reports/tickets?status=invalid-status", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("ListTickets(status invalide) = %d, attendu 400", w.Code)
	}
}

// ─── ListTickets — filtre valide → nil repo → panic → 500 ───────────────────

func TestListTickets_NilRepo_500(t *testing.T) {
	r := newRecoveryRouter(t)
	tok := makeReportToken(t, "moderator")
	req := httptest.NewRequest(http.MethodGet, "/reports/tickets", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListTickets(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── PendingWarnings — nil repo → 500 ────────────────────────────────────────

func TestPendingWarnings_NilRepo_500(t *testing.T) {
	r := newRecoveryRouter(t)
	tok := makeReportToken(t, "user")
	req := httptest.NewRequest(http.MethodGet, "/reports/warnings/pending", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("PendingWarnings(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── GetSettings — nil repo → 500 ────────────────────────────────────────────

func TestGetSettings_NilRepo_500(t *testing.T) {
	r := newRecoveryRouter(t)
	tok := makeReportToken(t, "moderator")
	req := httptest.NewRequest(http.MethodGet, "/reports/settings", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("GetSettings(nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── parseTime — couvert indirectement via ListTickets avec since/until ──────

func TestListTickets_SinceValide_NilRepo_500(t *testing.T) {
	r := newRecoveryRouter(t)
	tok := makeReportToken(t, "moderator")
	req := httptest.NewRequest(http.MethodGet, "/reports/tickets?since=2024-01-01T00:00:00Z", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// parseTime retourne la date → ListTickets appelle le repo (nil) → 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ListTickets(since valide, nil repo) = %d, attendu 500", w.Code)
	}
}

// ─── respondError — branches via service guards ───────────────────────────────

// ErrValidation → 400 (via Create reason invalide).
// ErrInvalidID → 400 (via GetTicket id invalide).
// default → 500 (via ListTickets nil repo).
// — branches couvertes par les tests ci-dessus.
