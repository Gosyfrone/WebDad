package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/webdad/report-service/internal/service"
)

// TestParseTime couvre les trois branches : vide → nil, invalide → nil, valide → date.
func TestParseTime(t *testing.T) {
	if parseTime("") != nil {
		t.Error("parseTime(\"\") devrait renvoyer nil")
	}
	if parseTime("pas-une-date") != nil {
		t.Error("parseTime(date invalide) devrait renvoyer nil")
	}
	ts := parseTime("2020-01-02T03:04:05Z")
	if ts == nil || !ts.Equal(time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Errorf("parseTime(date valide) = %v, attendu 2020-01-02T03:04:05Z", ts)
	}
}

// doCanceled exécute une requête authentifiée dont le contexte est DÉJÀ annulé :
// le middleware JWT passe (lit l'en-tête, pas le contexte), puis le handler
// appelle le service avec ce contexte → l'accès Mongo échoue avec une erreur
// inattendue (ni métier ni NotFound) → respondError tombe dans `default` (500).
// Exerce ainsi la branche 500 du handler ET le cas par défaut de respondError.
func doCanceled(t *testing.T, r *gin.Engine, method, path, role string) *httptest.ResponseRecorder {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(method, path, nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+makeReportToken(t, role))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHTTPErr_GetSettings_CtxAnnule_500(t *testing.T) {
	r := liveRouter(t) // skip si MONGO_TEST_URI absent
	if w := doCanceled(t, r, http.MethodGet, "/reports/settings", "moderator"); w.Code != http.StatusInternalServerError {
		t.Fatalf("GetSettings(ctx annulé) = %d, attendu 500 (%s)", w.Code, w.Body.String())
	}
}

func TestHTTPErr_PendingWarnings_CtxAnnule_500(t *testing.T) {
	r := liveRouter(t)
	if w := doCanceled(t, r, http.MethodGet, "/reports/warnings/pending", "user"); w.Code != http.StatusInternalServerError {
		t.Fatalf("PendingWarnings(ctx annulé) = %d, attendu 500 (%s)", w.Code, w.Body.String())
	}
}

// ─── Garde défensive : claims absents (branche `if !ok`) ─────────────────────
// En production JWTAuth garantit la présence des claims ; on appelle donc les
// handlers DIRECTEMENT, sans middleware, pour exercer leur garde 401. Le service
// n'est jamais atteint (retour avant la 1re ligne métier), d'où un service nil.

// ctxSansClaims construit un *gin.Context isolé (sans claims posés) pour un
// appel direct de handler, et renvoie le recorder associé.
func ctxSansClaims(method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	return c, w
}

func TestHandler_Create_SansClaims_401(t *testing.T) {
	h := NewReportHandler((*service.ReportService)(nil))
	c, w := ctxSansClaims(http.MethodPost, "/reports")
	h.Create(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Create sans claims = %d, attendu 401", w.Code)
	}
}

func TestHandler_PendingWarnings_SansClaims_401(t *testing.T) {
	h := NewReportHandler((*service.ReportService)(nil))
	c, w := ctxSansClaims(http.MethodGet, "/reports/warnings/pending")
	h.PendingWarnings(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("PendingWarnings sans claims = %d, attendu 401", w.Code)
	}
}
