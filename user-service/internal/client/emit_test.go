package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// ─── Emit avec un vrai serveur HTTP ──────────────────────────────────────────

func TestEmit_SuccèsHTTP(t *testing.T) {
	var mu sync.Mutex
	var received Event

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method != http.MethodPost {
			t.Errorf("méthode attendue POST, obtenu %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type attendu application/json, obtenu %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("X-Internal-Secret") != "my-secret" {
			t.Errorf("X-Internal-Secret attendu my-secret, obtenu %s", r.Header.Get("X-Internal-Secret"))
		}
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// NewNotificationClient ajoute /internal/events à l'URL, on retire le suffixe du srv.URL
	c := &NotificationClient{
		url:    srv.URL,
		secret: "my-secret",
		client: &http.Client{Timeout: 2 * time.Second},
	}

	ev := Event{Type: TypeFollow, ActorID: "actor-1", RecipientID: "recipient-1"}
	c.Emit(ev)

	// Emit est asynchrone (goroutine), on attend que le serveur reçoive
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		got := received
		mu.Unlock()
		if got.Type == TypeFollow {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if received.Type != TypeFollow {
		t.Fatalf("type reçu = %q, attendu %q", received.Type, TypeFollow)
	}
	if received.ActorID != "actor-1" {
		t.Fatalf("actor_id reçu = %q, attendu %q", received.ActorID, "actor-1")
	}
}

func TestEmit_ServeurErreur_NePasPlanter(t *testing.T) {
	// Serveur qui retourne 500 : Emit doit loguer et ignorer (ne pas paniquer).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := &NotificationClient{
		url:    srv.URL,
		secret: "s",
		client: &http.Client{Timeout: 1 * time.Second},
	}
	c.Emit(Event{Type: TypeFollowRequest, ActorID: "a", RecipientID: "b"})
	// Laisse la goroutine se terminer
	time.Sleep(200 * time.Millisecond)
}

func TestEmit_Retract_EnvoiOK(t *testing.T) {
	var mu sync.Mutex
	var received Event

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &NotificationClient{
		url:    srv.URL,
		secret: "s",
		client: &http.Client{Timeout: 2 * time.Second},
	}

	ev := Event{Type: TypeFollowRequest, ActorID: "a", RecipientID: "b", Retract: true}
	c.Emit(ev)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		got := received
		mu.Unlock()
		if got.Type == TypeFollowRequest {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if !received.Retract {
		t.Fatal("Retract devrait être true dans l'event reçu")
	}
}

func TestEmit_URLInjoignable_NePasPlanter(t *testing.T) {
	c := &NotificationClient{
		url:    "http://127.0.0.1:19998", // rien n'écoute
		secret: "s",
		client: &http.Client{Timeout: 50 * time.Millisecond},
	}
	c.Emit(Event{Type: TypeFollow, ActorID: "a", RecipientID: "b"})
	// Laisse la goroutine tenter et échouer
	time.Sleep(200 * time.Millisecond)
}
