package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ─── NewProfilClient ──────────────────────────────────────────────────────────

func TestNewProfilClient_Construit(t *testing.T) {
	c := NewProfilClient("http://profil-service:8083")
	if c.baseURL != "http://profil-service:8083" {
		t.Fatalf("baseURL = %q", c.baseURL)
	}
	if c.httpClient == nil {
		t.Fatal("httpClient nil")
	}
}

// ─── ProfilClient.Visibility ──────────────────────────────────────────────────

func TestVisibility_Succès(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("méthode attendue GET, obtenu %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(visibilityResponse{Visibility: "public"})
	}))
	defer srv.Close()

	c := NewProfilClient(srv.URL)
	vis, err := c.Visibility(context.Background(), "user-123")
	if err != nil {
		t.Fatalf("Visibility inattendue erreur : %v", err)
	}
	if vis != "public" {
		t.Fatalf("Visibility = %q, attendu %q", vis, "public")
	}
}

func TestVisibility_Privé(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(visibilityResponse{Visibility: VisibilityPrivate})
	}))
	defer srv.Close()

	c := NewProfilClient(srv.URL)
	vis, err := c.Visibility(context.Background(), "user-456")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if vis != VisibilityPrivate {
		t.Fatalf("Visibility = %q, attendu %q", vis, VisibilityPrivate)
	}
}

func TestVisibility_404_RetourneVide(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewProfilClient(srv.URL)
	vis, err := c.Visibility(context.Background(), "inconnu")
	if err != nil {
		t.Fatalf("404 ne doit pas retourner d'erreur : %v", err)
	}
	if vis != "" {
		t.Fatalf("Visibility pour 404 = %q, attendu vide", vis)
	}
}

func TestVisibility_ErreurServeur_RetourneErr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewProfilClient(srv.URL)
	_, err := c.Visibility(context.Background(), "user-789")
	if err == nil {
		t.Fatal("erreur 500 doit retourner une erreur")
	}
}

func TestVisibility_ErreurBadRequest_RetourneErr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c := NewProfilClient(srv.URL)
	_, err := c.Visibility(context.Background(), "user-bad")
	if err == nil {
		t.Fatal("erreur 400 doit retourner une erreur")
	}
}

func TestVisibility_JSONInvalide_RetourneErr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewProfilClient(srv.URL)
	_, err := c.Visibility(context.Background(), "user-json")
	if err == nil {
		t.Fatal("JSON invalide doit retourner une erreur")
	}
}

func TestVisibility_ServeurInjoignable_RetourneErr(t *testing.T) {
	c := NewProfilClient("http://127.0.0.1:19999") // rien n'écoute là
	c.httpClient = &http.Client{Timeout: 50 * time.Millisecond}
	_, err := c.Visibility(context.Background(), "x")
	if err == nil {
		t.Fatal("serveur injoignable doit retourner une erreur")
	}
}

func TestVisibility_ContextAnnulé_RetourneErr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewProfilClient(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // annulé immédiatement
	_, err := c.Visibility(ctx, "user-ctx")
	if err == nil {
		t.Fatal("contexte annulé doit retourner une erreur")
	}
}

// ─── VisibilityPrivate constant ───────────────────────────────────────────────

func TestVisibilityPrivateConstante(t *testing.T) {
	if VisibilityPrivate != "private" {
		t.Fatalf("VisibilityPrivate = %q, attendu %q", VisibilityPrivate, "private")
	}
}
