package userdir

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	r := New("http://user-service:8082")
	if r == nil {
		t.Fatal("New() = nil")
	}
	if r.base != "http://user-service:8082" {
		t.Fatalf("base = %q, attendu http://user-service:8082", r.base)
	}
}

func TestResolveHandle_Succès(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/by-username/alice" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{"id": "uuid-alice-123"},
		})
	}))
	defer srv.Close()

	resolver := New(srv.URL)
	id, ok, err := resolver.ResolveHandle(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ResolveHandle erreur : %v", err)
	}
	if !ok {
		t.Fatal("ok = false, attendu true")
	}
	if id != "uuid-alice-123" {
		t.Fatalf("id = %q, attendu uuid-alice-123", id)
	}
}

func TestResolveHandle_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	resolver := New(srv.URL)
	id, ok, err := resolver.ResolveHandle(context.Background(), "unknown")
	if err != nil {
		t.Fatalf("ResolveHandle 404 erreur inattendue : %v", err)
	}
	if ok {
		t.Fatal("ok = true sur 404, attendu false")
	}
	if id != "" {
		t.Fatalf("id = %q, attendu ''", id)
	}
}

func TestResolveHandle_ErreurServeur(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	resolver := New(srv.URL)
	_, _, err := resolver.ResolveHandle(context.Background(), "alice")
	if err == nil {
		t.Fatal("500 doit retourner une erreur")
	}
}

func TestResolveHandle_JSONInvalide(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	resolver := New(srv.URL)
	_, _, err := resolver.ResolveHandle(context.Background(), "alice")
	if err == nil {
		t.Fatal("JSON invalide doit retourner une erreur")
	}
}

func TestResolveHandle_IDVide(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{"id": ""},
		})
	}))
	defer srv.Close()

	resolver := New(srv.URL)
	id, ok, err := resolver.ResolveHandle(context.Background(), "alice")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if ok {
		t.Fatal("id vide doit retourner ok=false")
	}
	if id != "" {
		t.Fatalf("id = %q, attendu ''", id)
	}
}

func TestResolveHandle_ServeDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	resolver := New(srv.URL)
	_, _, err := resolver.ResolveHandle(context.Background(), "alice")
	if err == nil {
		t.Fatal("serveur fermé doit retourner une erreur")
	}
}
