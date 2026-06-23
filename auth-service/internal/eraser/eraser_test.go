package eraser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestEraseTolerates404AndReportsFailures : un 404 est toléré (succès), un 5xx
// remonte comme échec ; le bearer est transmis.
func TestEraseTolerates404AndReportsFailures(t *testing.T) {
	gotAuth := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		switch {
		case strings.HasSuffix(r.URL.Path, "/hard"): // étape "user" → échec serveur
			w.WriteHeader(http.StatusInternalServerError)
		case strings.Contains(r.URL.Path, "/profils/"): // profil → 404 toléré
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	e := New(Targets{User: srv.URL, Profil: srv.URL, Post: srv.URL, Message: srv.URL, Media: srv.URL})
	failed := e.Erase(context.Background(), "tok123", "u1")

	if len(failed) != 1 || failed[0] != "user" {
		t.Fatalf("échecs attendus [user], eu %v", failed)
	}
	if gotAuth != "Bearer tok123" {
		t.Fatalf("bearer non transmis : %q", gotAuth)
	}
}

// TestEraseSkipsUnconfigured : un service sans URL est ignoré (aucun échec).
func TestEraseSkipsUnconfigured(t *testing.T) {
	e := New(Targets{})
	if failed := e.Erase(context.Background(), "tok", "u1"); len(failed) != 0 {
		t.Fatalf("aucun échec attendu (rien de configuré), eu %v", failed)
	}
}

func TestDeleteOKRejectsMalformedURL(t *testing.T) {
	e := New(Targets{})
	if e.deleteOK(context.Background(), "token", "://bad-url") {
		t.Fatal("malformed URL should fail")
	}
}
