package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/webdad/message-service/internal/realtime"
	"github.com/webdad/message-service/internal/service"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// --- Tests de respondError ---

func TestRespondError_ErrConversationNotFound(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrConversationNotFound)
	if w.Code != http.StatusNotFound {
		t.Errorf("ErrConversationNotFound → %d, attendu 404", w.Code)
	}
}

func TestRespondError_ErrKeyNotFound(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrKeyNotFound)
	if w.Code != http.StatusNotFound {
		t.Errorf("ErrKeyNotFound → %d, attendu 404", w.Code)
	}
}

func TestRespondError_ErrBackupNotFound(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrBackupNotFound)
	if w.Code != http.StatusNotFound {
		t.Errorf("ErrBackupNotFound → %d, attendu 404", w.Code)
	}
}

func TestRespondError_ErrMessageNotFound(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrMessageNotFound)
	if w.Code != http.StatusNotFound {
		t.Errorf("ErrMessageNotFound → %d, attendu 404", w.Code)
	}
}

func TestRespondError_ErrTargetNotMember(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrTargetNotMember)
	if w.Code != http.StatusNotFound {
		t.Errorf("ErrTargetNotMember → %d, attendu 404", w.Code)
	}
}

func TestRespondError_ErrInvalidID(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrInvalidID)
	if w.Code != http.StatusBadRequest {
		t.Errorf("ErrInvalidID → %d, attendu 400", w.Code)
	}
}

func TestRespondError_ErrSelfConversation(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrSelfConversation)
	if w.Code != http.StatusBadRequest {
		t.Errorf("ErrSelfConversation → %d, attendu 400", w.Code)
	}
}

func TestRespondError_ErrMissingEnvelope(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrMissingEnvelope)
	if w.Code != http.StatusBadRequest {
		t.Errorf("ErrMissingEnvelope → %d, attendu 400", w.Code)
	}
}

func TestRespondError_ErrInvalidGroup(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrInvalidGroup)
	if w.Code != http.StatusBadRequest {
		t.Errorf("ErrInvalidGroup → %d, attendu 400", w.Code)
	}
}

func TestRespondError_ErrInvalidCommunity(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrInvalidCommunity)
	if w.Code != http.StatusBadRequest {
		t.Errorf("ErrInvalidCommunity → %d, attendu 400", w.Code)
	}
}

func TestRespondError_ErrInvalidRole(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrInvalidRole)
	if w.Code != http.StatusBadRequest {
		t.Errorf("ErrInvalidRole → %d, attendu 400", w.Code)
	}
}

func TestRespondError_ErrNotMember(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrNotMember)
	if w.Code != http.StatusForbidden {
		t.Errorf("ErrNotMember → %d, attendu 403", w.Code)
	}
}

func TestRespondError_ErrCannotWrite(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrCannotWrite)
	if w.Code != http.StatusForbidden {
		t.Errorf("ErrCannotWrite → %d, attendu 403", w.Code)
	}
}

func TestRespondError_ErrOwnerOnly(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrOwnerOnly)
	if w.Code != http.StatusForbidden {
		t.Errorf("ErrOwnerOnly → %d, attendu 403", w.Code)
	}
}

func TestRespondError_ErrOwnerCannotLeave(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrOwnerCannotLeave)
	if w.Code != http.StatusForbidden {
		t.Errorf("ErrOwnerCannotLeave → %d, attendu 403", w.Code)
	}
}

func TestRespondError_ErrNotMessageOwner(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrNotMessageOwner)
	if w.Code != http.StatusForbidden {
		t.Errorf("ErrNotMessageOwner → %d, attendu 403", w.Code)
	}
}

func TestRespondError_ErrCannotDelete(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrCannotDelete)
	if w.Code != http.StatusForbidden {
		t.Errorf("ErrCannotDelete → %d, attendu 403", w.Code)
	}
}

func TestRespondError_ErrNotGroup(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrNotGroup)
	if w.Code != http.StatusConflict {
		t.Errorf("ErrNotGroup → %d, attendu 409", w.Code)
	}
}

func TestRespondError_ErrNotCommunity(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrNotCommunity)
	if w.Code != http.StatusConflict {
		t.Errorf("ErrNotCommunity → %d, attendu 409", w.Code)
	}
}

func TestRespondError_ErrNotManageable(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrNotManageable)
	if w.Code != http.StatusConflict {
		t.Errorf("ErrNotManageable → %d, attendu 409", w.Code)
	}
}

func TestRespondError_ErrAlreadyMember(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrAlreadyMember)
	if w.Code != http.StatusConflict {
		t.Errorf("ErrAlreadyMember → %d, attendu 409", w.Code)
	}
}

func TestRespondError_ErrTalkersFull(t *testing.T) {
	w, c := newGinContext()
	respondError(c, service.ErrTalkersFull)
	if w.Code != http.StatusConflict {
		t.Errorf("ErrTalkersFull → %d, attendu 409", w.Code)
	}
}

func TestRespondError_InternalError(t *testing.T) {
	w, c := newGinContext()
	import_err := newUnexpectedError()
	respondError(c, import_err)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("erreur inconnue → %d, attendu 500", w.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err == nil {
		if body["error"] != "erreur interne" {
			t.Errorf("corps = %v, attendu erreur interne", body)
		}
	}
}

func newUnexpectedError() error {
	return fmt.Errorf("une erreur technique inconnue")
}

func TestRespondError_InternalError_Unknown(t *testing.T) {
	// Une erreur inconnue du service doit retourner 500
	w, c := newGinContext()
	import_unknown_err := unknownErr("une erreur technique inconnue")
	respondError(c, import_unknown_err)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("erreur inconnue → %d, attendu 500", w.Code)
	}
}

type unknownErr string

func (e unknownErr) Error() string { return string(e) }

// --- Tests de excluding ---

func TestExcluding_Basic(t *testing.T) {
	ids := []string{"u1", "u2", "u3"}
	got := excluding(ids, "u2")
	if len(got) != 2 {
		t.Fatalf("excluding = %v, attendu [u1 u3]", got)
	}
	if got[0] != "u1" || got[1] != "u3" {
		t.Fatalf("excluding = %v, attendu [u1 u3]", got)
	}
}

func TestExcluding_NotPresent(t *testing.T) {
	ids := []string{"u1", "u2"}
	got := excluding(ids, "u99")
	if len(got) != 2 {
		t.Fatalf("excluding(absent) = %v, attendu tous les membres", got)
	}
}

func TestExcluding_EmptyList(t *testing.T) {
	got := excluding(nil, "u1")
	if len(got) != 0 {
		t.Fatalf("excluding(nil) = %v, attendu vide", got)
	}
}

func TestExcluding_SingleElement(t *testing.T) {
	got := excluding([]string{"u1"}, "u1")
	if len(got) != 0 {
		t.Fatalf("excluding([u1], u1) = %v, attendu vide", got)
	}
}

// --- Tests de pageLimit ---

func TestPageLimit_Valid(t *testing.T) {
	cases := []struct {
		queryVal string
		want     int64
	}{
		{"10", 10},
		{"50", 50},
		{"100", 100},
		{"", 0},    // valeur par défaut → 0 (le service applique le defaut)
		{"abc", 0}, // invalide → 0
		{"-5", -5}, // négatif passé tel quel (le service borne)
	}

	for _, tc := range cases {
		r := httptest.NewRequest(http.MethodGet, "/?limit="+tc.queryVal, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = r
		got := pageLimit(c)
		if got != tc.want {
			t.Errorf("pageLimit(%q) = %d, attendu %d", tc.queryVal, got, tc.want)
		}
	}
}

// --- Tests de pageOffset ---

func TestPageOffset_Valid(t *testing.T) {
	cases := []struct {
		queryVal string
		want     int64
	}{
		{"0", 0},
		{"20", 20},
		{"", 0},
		{"bad", 0},
	}

	for _, tc := range cases {
		r := httptest.NewRequest(http.MethodGet, "/?offset="+tc.queryVal, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = r
		got := pageOffset(c)
		if got != tc.want {
			t.Errorf("pageOffset(%q) = %d, attendu %d", tc.queryVal, got, tc.want)
		}
	}
}

// --- Tests de broadcastReceipt ---

func TestBroadcastReceipt_EmptyTargets(t *testing.T) {
	hub := realtime.NewHub()
	h := &ConversationHandler{hub: hub}
	w := httptest.NewRecorder()
	_, _ = gin.CreateTestContext(w)
	// Aucune cible → ne doit pas paniquer
	h.broadcastReceipt(nil, "conv1", "u1", nil, nil)
}

func TestBroadcastReceipt_WithDeliveredAndRead(t *testing.T) {
	hub := realtime.NewHub()
	h := &ConversationHandler{hub: hub}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	_ = c
	now := time.Now()
	// Avec des cibles mais pas de connexion active → ne doit pas paniquer
	h.broadcastReceipt([]string{"u2", "u3"}, "conv1", "u1", &now, &now)
}

func TestBroadcastReceipt_OnlyDelivered(t *testing.T) {
	hub := realtime.NewHub()
	h := &ConversationHandler{hub: hub}
	now := time.Now()
	h.broadcastReceipt([]string{"u2"}, "conv1", "u1", &now, nil)
}

func TestBroadcastReceipt_OnlyRead(t *testing.T) {
	hub := realtime.NewHub()
	h := &ConversationHandler{hub: hub}
	now := time.Now()
	h.broadcastReceipt([]string{"u2"}, "conv1", "u1", nil, &now)
}

// --- Helpers ---

func newGinContext() (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return w, c
}
