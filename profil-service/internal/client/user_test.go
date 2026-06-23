package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ─── NewUserClient ────────────────────────────────────────────────────────────

func TestNewUserClient_SupprimeSlashFinal(t *testing.T) {
	c := NewUserClient("http://user-service:8082/")
	if c.baseURL != "http://user-service:8082" {
		t.Fatalf("baseURL = %q, attendu sans slash final", c.baseURL)
	}
}

func TestNewUserClient_SansSlashFinal(t *testing.T) {
	c := NewUserClient("http://user-service:8082")
	if c.baseURL != "http://user-service:8082" {
		t.Fatalf("baseURL = %q", c.baseURL)
	}
	if c.http == nil {
		t.Fatal("http client nil")
	}
}

// ─── AcceptAllFollowRequests ──────────────────────────────────────────────────

func TestAcceptAllFollowRequests_Succès(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewUserClient(srv.URL)
	err := c.AcceptAllFollowRequests(context.Background(), "owner-123")
	if err != nil {
		t.Fatalf("AcceptAllFollowRequests: %v", err)
	}
}

func TestAcceptAllFollowRequests_Erreur(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewUserClient(srv.URL)
	err := c.AcceptAllFollowRequests(context.Background(), "owner-123")
	if err == nil {
		t.Fatal("status 500 devrait retourner une erreur")
	}
}

func TestAcceptAllFollowRequests_URLInvalide(t *testing.T) {
	c := NewUserClient("%")
	if err := c.AcceptAllFollowRequests(context.Background(), "owner-123"); err == nil {
		t.Fatal("URL invalide devrait retourner une erreur")
	}
}

// ─── IsFollowing ─────────────────────────────────────────────────────────────

func TestIsFollowing_True(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]bool{"isFollowing": true})
	}))
	defer srv.Close()

	c := NewUserClient(srv.URL)
	ok, err := c.IsFollowing(context.Background(), "a", "b")
	if err != nil {
		t.Fatalf("IsFollowing: %v", err)
	}
	if !ok {
		t.Fatal("attendu true")
	}
}

func TestIsFollowing_False(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]bool{"isFollowing": false})
	}))
	defer srv.Close()

	c := NewUserClient(srv.URL)
	ok, err := c.IsFollowing(context.Background(), "a", "b")
	if err != nil {
		t.Fatalf("IsFollowing: %v", err)
	}
	if ok {
		t.Fatal("attendu false")
	}
}

func TestIsFollowing_ErreurServeur(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := NewUserClient(srv.URL)
	_, err := c.IsFollowing(context.Background(), "a", "b")
	if err == nil {
		t.Fatal("status 502 devrait retourner une erreur")
	}
}

func TestIsFollowing_URLInvalide(t *testing.T) {
	c := NewUserClient("%")
	if _, err := c.IsFollowing(context.Background(), "a", "b"); err == nil {
		t.Fatal("URL invalide devrait retourner une erreur")
	}
}

func TestIsFollowing_JSONInvalide(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{invalid}`))
	}))
	defer srv.Close()

	c := NewUserClient(srv.URL)
	if _, err := c.IsFollowing(context.Background(), "a", "b"); err == nil {
		t.Fatal("JSON invalide devrait retourner une erreur")
	}
}
