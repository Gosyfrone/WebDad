package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/webdad/mail-service/internal/mailer"
)

type noopMailer struct{}

func (noopMailer) Send(mailer.Message) error { return nil }

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, "mail-service", noopMailer{}, "internal-secret")
	return r
}

func TestRoutesRegister(t *testing.T) {
	r := newTestRouter()
	if len(r.Routes()) == 0 {
		t.Fatal("aucune route enregistrée")
	}
}

func TestHealth(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /health = %d, attendu 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ok") {
		t.Fatalf("body = %s, attendu 'ok'", w.Body.String())
	}
}

func TestSend_SecretInvalide(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodPost, "/internal/send",
		strings.NewReader(`{"to":"a@b.com","subject":"s","text":"t"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", "wrong-secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("mauvais secret = %d, attendu 401", w.Code)
	}
}

func TestSend_SecretOK_Succès(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodPost, "/internal/send",
		strings.NewReader(`{"to":"a@b.com","subject":"s","text":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", "internal-secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("envoi OK = %d, attendu 202", w.Code)
	}
}

func TestSend_CorpsInvalide(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodPost, "/internal/send",
		strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", "internal-secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("corps invalide = %d, attendu 400", w.Code)
	}
}

func TestSend_HTMLEtTextVides(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodPost, "/internal/send",
		strings.NewReader(`{"to":"a@b.com","subject":"s"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", "internal-secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("sans html/text = %d, attendu 400", w.Code)
	}
}
