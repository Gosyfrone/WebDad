package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ─── NoopPostModerator ────────────────────────────────────────────────────────

func TestNoopPostModerator_AutoHide_NePasPlanter(t *testing.T) {
	var n NoopPostModerator
	n.AutoHide("any-post-id")
}

func TestNoopPostModerator_AutoUnhide_NePasPlanter(t *testing.T) {
	var n NoopPostModerator
	n.AutoUnhide("any-post-id")
}

// ─── NewPostClient ────────────────────────────────────────────────────────────

func TestNewPostClient_ChampsCorrects(t *testing.T) {
	c := NewPostClient("http://post-service:8084", "shared-secret")
	if c.baseURL != "http://post-service:8084" {
		t.Fatalf("baseURL = %q", c.baseURL)
	}
	if c.secret != "shared-secret" {
		t.Fatalf("secret = %q", c.secret)
	}
	if c.client == nil {
		t.Fatal("http client nil")
	}
}

// ─── AutoHide / AutoUnhide — fire-and-forget (goroutine) ─────────────────────
// call() lance une goroutine, donc on attend le signal du handler.

func TestHTTPPostClient_AutoHide_AvecServeur(t *testing.T) {
	done := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		done <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewPostClient(srv.URL, "s")
	c.AutoHide("post-abc")

	select {
	case path := <-done:
		if !strings.Contains(path, "post-abc") {
			t.Fatalf("path = %q, attendu 'post-abc'", path)
		}
		if !strings.HasSuffix(path, "auto-hide") {
			t.Fatalf("path ne se termine pas par 'auto-hide' : %q", path)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout : AutoHide n'a pas déclenché de requête HTTP")
	}
}

func TestHTTPPostClient_AutoUnhide_AvecServeur(t *testing.T) {
	done := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		done <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewPostClient(srv.URL, "s")
	c.AutoUnhide("post-xyz")

	select {
	case path := <-done:
		if !strings.HasSuffix(path, "auto-unhide") {
			t.Fatalf("path = %q, attendu suffix 'auto-unhide'", path)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout : AutoUnhide n'a pas déclenché de requête HTTP")
	}
}

func TestHTTPPostClient_AutoHide_IDVide_PasDeRequête(t *testing.T) {
	called := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewPostClient(srv.URL, "s")
	c.AutoHide("")

	// 50ms sans appel → OK
	select {
	case <-called:
		t.Fatal("id vide ne devrait pas déclencher de requête HTTP")
	case <-time.After(50 * time.Millisecond):
	}
}
