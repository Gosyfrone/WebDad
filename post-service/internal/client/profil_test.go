package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewProfilClient(t *testing.T) {
	c := NewProfilClient("http://profil-service:8083")
	if c == nil {
		t.Fatal("NewProfilClient() = nil")
	}
}

func TestVisibility_Public(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(visibilityResponse{Visibility: "public"})
	}))
	defer srv.Close()

	c := NewProfilClient(srv.URL)
	vis, err := c.Visibility(context.Background(), "user-id")
	if err != nil {
		t.Fatalf("Visibility erreur : %v", err)
	}
	if vis != "public" {
		t.Fatalf("Visibility = %q, attendu public", vis)
	}
}

func TestVisibility_Private(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(visibilityResponse{Visibility: "private"})
	}))
	defer srv.Close()

	c := NewProfilClient(srv.URL)
	vis, err := c.Visibility(context.Background(), "user-id")
	if err != nil {
		t.Fatalf("Visibility erreur : %v", err)
	}
	if vis != "private" {
		t.Fatalf("Visibility = %q, attendu private", vis)
	}
}

func TestVisibility_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := NewProfilClient(srv.URL)
	vis, err := c.Visibility(context.Background(), "missing-id")
	if err != nil {
		t.Fatalf("404 doit retourner public (fallback) : %v", err)
	}
	if vis != VisibilityPublic {
		t.Fatalf("Visibility 404 = %q, attendu %q", vis, VisibilityPublic)
	}
}

func TestVisibility_ErreurServeur(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewProfilClient(srv.URL)
	_, err := c.Visibility(context.Background(), "uid")
	if err == nil {
		t.Fatal("500 doit retourner une erreur")
	}
}

func TestVisibility_ServeDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	c := NewProfilClient(srv.URL)
	_, err := c.Visibility(context.Background(), "uid")
	if err == nil {
		t.Fatal("serveur fermé doit retourner une erreur")
	}
}
