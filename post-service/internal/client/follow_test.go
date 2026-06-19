package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewFollowClient(t *testing.T) {
	c := NewFollowClient("http://user-service:8082", "secret")
	if c == nil {
		t.Fatal("NewFollowClient() = nil")
	}
}

func TestIsFollowing_True(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Secret") != "mysecret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(followResponse{IsFollowing: true})
	}))
	defer srv.Close()

	c := NewFollowClient(srv.URL, "mysecret")
	ok, err := c.IsFollowing(context.Background(), "follower-id", "following-id")
	if err != nil {
		t.Fatalf("IsFollowing erreur : %v", err)
	}
	if !ok {
		t.Fatal("IsFollowing = false, attendu true")
	}
}

func TestIsFollowing_False(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(followResponse{IsFollowing: false})
	}))
	defer srv.Close()

	c := NewFollowClient(srv.URL, "secret")
	ok, err := c.IsFollowing(context.Background(), "a", "b")
	if err != nil {
		t.Fatalf("IsFollowing erreur : %v", err)
	}
	if ok {
		t.Fatal("IsFollowing = true, attendu false")
	}
}

func TestIsFollowing_ErreurServeur(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewFollowClient(srv.URL, "secret")
	_, err := c.IsFollowing(context.Background(), "a", "b")
	if err == nil {
		t.Fatal("500 doit retourner une erreur")
	}
}

func TestIsFollowing_ServeDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	c := NewFollowClient(srv.URL, "secret")
	_, err := c.IsFollowing(context.Background(), "a", "b")
	if err == nil {
		t.Fatal("serveur fermé doit retourner une erreur")
	}
}
