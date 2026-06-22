package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/webdad/report-service/internal/repository"
	"github.com/webdad/report-service/internal/service"
	"github.com/webdad/report-service/internal/testutil"
)

// liveRouter monte les routes avec un service ADOSSÉ à un Mongo réel (ou skip).
func liveRouter(t *testing.T) *gin.Engine {
	t.Helper()
	db := testutil.MongoDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	svc := service.NewReportService(repository.NewReportRepository(db), nil) // NoopPostModerator
	RegisterRoutes(r, "report-service", svc, testSecret)
	return r
}

// do exécute une requête authentifiée et renvoie le recorder.
func do(t *testing.T, r *gin.Engine, method, path, role, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rd *strings.Reader
	if body != "" {
		rd = strings.NewReader(body)
	} else {
		rd = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Authorization", "Bearer "+makeReportToken(t, role))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ticketID dépose un signalement de modération et renvoie l'id du ticket créé.
func ticketID(t *testing.T, r *gin.Engine, entityID string) string {
	t.Helper()
	w := do(t, r, http.MethodPost, "/reports", "user",
		`{"category":"moderation","reason":"spam","entity_type":"post","entity_id":"`+entityID+`","entity_owner_id":"owner"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("création signalement = %d (%s)", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode ticket : %v", err)
	}
	return resp.Data.ID
}

// ─── Parcours complet via HTTP ───────────────────────────────────────────────

func TestHTTP_CreateAndList(t *testing.T) {
	r := liveRouter(t)
	_ = ticketID(t, r, "postA")

	w := do(t, r, http.MethodGet, "/reports/tickets?category=moderation", "moderator", "")
	if w.Code != http.StatusOK {
		t.Fatalf("ListTickets = %d (%s)", w.Code, w.Body.String())
	}
}

func TestHTTP_GetTicket_OKEt404(t *testing.T) {
	r := liveRouter(t)
	id := ticketID(t, r, "postB")

	if w := do(t, r, http.MethodGet, "/reports/tickets/"+id, "moderator", ""); w.Code != http.StatusOK {
		t.Fatalf("GetTicket = %d (%s)", w.Code, w.Body.String())
	}
	if w := do(t, r, http.MethodGet, "/reports/tickets/507f1f77bcf86cd799439011", "moderator", ""); w.Code != http.StatusNotFound {
		t.Fatalf("GetTicket(absent) = %d, attendu 404", w.Code)
	}
}

func TestHTTP_ReplyStatusRemovalApprove(t *testing.T) {
	r := liveRouter(t)
	id := ticketID(t, r, "postC")

	if w := do(t, r, http.MethodPost, "/reports/tickets/"+id+"/replies", "moderator", `{"text":"réponse"}`); w.Code != http.StatusOK {
		t.Fatalf("Reply = %d (%s)", w.Code, w.Body.String())
	}
	if w := do(t, r, http.MethodPatch, "/reports/tickets/"+id+"/status", "moderator", `{"status":"closed"}`); w.Code != http.StatusOK {
		t.Fatalf("ChangeStatus = %d (%s)", w.Code, w.Body.String())
	}
	if w := do(t, r, http.MethodPost, "/reports/tickets/"+id+"/removal", "moderator", ""); w.Code != http.StatusOK {
		t.Fatalf("RecordRemoval = %d (%s)", w.Code, w.Body.String())
	}
	// Approve après removal → 409 (sanction déjà posée).
	if w := do(t, r, http.MethodPost, "/reports/tickets/"+id+"/approve", "moderator", ""); w.Code != http.StatusConflict {
		t.Fatalf("Approve après removal = %d, attendu 409", w.Code)
	}
}

func TestHTTP_Approve_OK(t *testing.T) {
	r := liveRouter(t)
	id := ticketID(t, r, "postD")
	if w := do(t, r, http.MethodPost, "/reports/tickets/"+id+"/approve", "moderator", ""); w.Code != http.StatusOK {
		t.Fatalf("Approve = %d (%s)", w.Code, w.Body.String())
	}
}

func TestHTTP_Transfer_Admin(t *testing.T) {
	r := liveRouter(t)
	// Crée un ticket de bug puis le transfère (admin).
	w := do(t, r, http.MethodPost, "/reports", "user", `{"category":"bug","reason":"bug","text":"plante"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("création bug = %d (%s)", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if tw := do(t, r, http.MethodPost, "/reports/tickets/"+resp.Data.ID+"/transfer", "admin", ""); tw.Code != http.StatusOK {
		t.Fatalf("Transfer = %d (%s)", tw.Code, tw.Body.String())
	}
}

func TestHTTP_Settings(t *testing.T) {
	r := liveRouter(t)
	if w := do(t, r, http.MethodGet, "/reports/settings", "moderator", ""); w.Code != http.StatusOK {
		t.Fatalf("GetSettings = %d (%s)", w.Code, w.Body.String())
	}
	if w := do(t, r, http.MethodPatch, "/reports/settings", "admin", `{"auto_hide_threshold":7}`); w.Code != http.StatusOK {
		t.Fatalf("UpdateSettings = %d (%s)", w.Code, w.Body.String())
	}
}

func TestHTTP_Warnings(t *testing.T) {
	r := liveRouter(t)
	// Émission (modérateur).
	if w := do(t, r, http.MethodPost, "/reports/warnings", "moderator",
		`{"target_user_id":"victim","message":"attention"}`); w.Code != http.StatusCreated {
		t.Fatalf("IssueWarning = %d (%s)", w.Code, w.Body.String())
	}
	// Profil de risque (modérateur).
	if w := do(t, r, http.MethodGet, "/reports/users/victim/warnings/count", "moderator", ""); w.Code != http.StatusOK {
		t.Fatalf("UserWarningCount = %d (%s)", w.Code, w.Body.String())
	}
}

func TestHTTP_PendingAndAckWarning(t *testing.T) {
	r := liveRouter(t)
	// La victime est l'utilisateur du token (même UserID dans makeToken).
	if w := do(t, r, http.MethodPost, "/reports/warnings", "moderator",
		`{"target_user_id":"11111111-1111-1111-1111-111111111111","message":"attention"}`); w.Code != http.StatusCreated {
		t.Fatalf("IssueWarning = %d (%s)", w.Code, w.Body.String())
	}

	w := do(t, r, http.MethodGet, "/reports/warnings/pending", "user", "")
	if w.Code != http.StatusOK {
		t.Fatalf("PendingWarnings = %d (%s)", w.Code, w.Body.String())
	}
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || len(resp.Data) == 0 {
		t.Fatalf("decode pending : %v (n=%d)", err, len(resp.Data))
	}
	if aw := do(t, r, http.MethodPost, "/reports/warnings/"+resp.Data[0].ID+"/ack", "user", ""); aw.Code != http.StatusNoContent {
		t.Fatalf("AckWarning = %d, attendu 204", aw.Code)
	}
}
