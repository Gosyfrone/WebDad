package handler

import (
	"net/http"
	"testing"
)

// Chemins d'erreur métier traversant respondError (404 / 409 / 400) avec un
// service adossé à un Mongo réel.

const absentID = "507f1f77bcf86cd799439011"

func TestHTTPErr_GetAbsent_404(t *testing.T) {
	r := liveRouter(t)
	for _, p := range []struct{ method, path string }{
		{http.MethodPost, "/reports/tickets/" + absentID + "/removal"},
		{http.MethodPost, "/reports/tickets/" + absentID + "/approve"},
		{http.MethodPost, "/reports/tickets/" + absentID + "/replies"},
		{http.MethodPatch, "/reports/tickets/" + absentID + "/status"},
	} {
		body := ""
		if p.path[len(p.path)-1] == 's' { // replies
			body = `{"text":"x"}`
		}
		if p.method == http.MethodPatch {
			body = `{"status":"open"}`
		}
		w := do(t, r, p.method, p.path, "moderator", body)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s %s = %d, attendu 404 (%s)", p.method, p.path, w.Code, w.Body.String())
		}
	}
}

func TestHTTPErr_Transfer_NonBug_409(t *testing.T) {
	r := liveRouter(t)
	id := ticketID(t, r, "postT") // ticket de modération
	w := do(t, r, http.MethodPost, "/reports/tickets/"+id+"/transfer", "admin", "")
	if w.Code != http.StatusConflict {
		t.Fatalf("Transfer(modération) = %d, attendu 409 (%s)", w.Code, w.Body.String())
	}
}

func TestHTTPErr_Transfer_Absent_404(t *testing.T) {
	r := liveRouter(t)
	w := do(t, r, http.MethodPost, "/reports/tickets/"+absentID+"/transfer", "admin", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("Transfer(absent) = %d, attendu 404 (%s)", w.Code, w.Body.String())
	}
}

func TestHTTPErr_CreateDuplicate_409(t *testing.T) {
	r := liveRouter(t)
	body := `{"category":"moderation","reason":"spam","entity_type":"post","entity_id":"dup","entity_owner_id":"o"}`
	if w := do(t, r, http.MethodPost, "/reports", "user", body); w.Code != http.StatusCreated {
		t.Fatalf("1er signalement = %d", w.Code)
	}
	if w := do(t, r, http.MethodPost, "/reports", "user", body); w.Code != http.StatusConflict {
		t.Fatalf("doublon = %d, attendu 409", w.Code)
	}
}

func TestHTTPErr_UpdateSettings_Negatif_400(t *testing.T) {
	r := liveRouter(t)
	w := do(t, r, http.MethodPatch, "/reports/settings", "admin", `{"auto_hide_threshold":-3}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("UpdateSettings(négatif) = %d, attendu 400 (%s)", w.Code, w.Body.String())
	}
}

func TestHTTPErr_IssueWarning_MessageTropLong_400(t *testing.T) {
	r := liveRouter(t)
	long := make([]byte, 501)
	for i := range long {
		long[i] = 'x'
	}
	body := `{"target_user_id":"v","message":"` + string(long) + `"}`
	w := do(t, r, http.MethodPost, "/reports/warnings", "moderator", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("IssueWarning(message trop long) = %d, attendu 400", w.Code)
	}
}

// AckWarning avec un id mal formé → ErrInvalidID → 400 (branche d'erreur du
// handler AckWarning).
func TestHTTPErr_AckWarning_IDInvalide_400(t *testing.T) {
	r := liveRouter(t)
	w := do(t, r, http.MethodPost, "/reports/warnings/pas-un-hex/ack", "user", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("AckWarning(id invalide) = %d, attendu 400 (%s)", w.Code, w.Body.String())
	}
}

// UserWarningCount avec un id vide (après trim) → ErrValidation → 400.
func TestHTTPErr_UserWarningCount_IDVide_400(t *testing.T) {
	r := liveRouter(t)
	w := do(t, r, http.MethodGet, "/reports/users/%20/warnings/count", "moderator", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("UserWarningCount(id vide) = %d, attendu 400 (%s)", w.Code, w.Body.String())
	}
}

// ListTickets avec tous les filtres → couvre min_reports (strconv), limit et les
// deux bornes temporelles (parseTime since + until).
func TestHTTP_ListTickets_TousFiltres(t *testing.T) {
	r := liveRouter(t)
	_ = ticketID(t, r, "postF")
	path := "/reports/tickets?category=moderation&status=open&min_reports=1&limit=10" +
		"&since=2020-01-01T00:00:00Z&until=2999-01-01T00:00:00Z"
	w := do(t, r, http.MethodGet, path, "moderator", "")
	if w.Code != http.StatusOK {
		t.Fatalf("ListTickets(filtres) = %d, attendu 200 (%s)", w.Code, w.Body.String())
	}
}
