package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewNotificationClient_EmptyBaseURL(t *testing.T) {
	if c := NewNotificationClient("", "secret"); c != nil {
		t.Fatalf("client = %#v, attendu nil", c)
	}
}

func TestEmitIdentityUpdated_Success(t *testing.T) {
	got := make(chan IdentityEvent, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/events" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("X-Internal-Secret") != "secret" {
			t.Fatalf("secret = %q", r.Header.Get("X-Internal-Secret"))
		}
		var ev IdentityEvent
		if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
			t.Fatalf("json: %v", err)
		}
		got <- ev
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	NewNotificationClient(srv.URL, "secret").EmitIdentityUpdated("u1", "political", "moderator")

	select {
	case ev := <-got:
		if ev.Type != TypeIdentityUpdated || ev.ActorID != "u1" || ev.TargetUserID != "u1" ||
			ev.Certification != "political" || ev.Role != "moderator" {
			t.Fatalf("event = %#v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("event non reçu")
	}
}

func TestEmitIdentityUpdated_BestEffortErrors(t *testing.T) {
	(*NotificationClient)(nil).EmitIdentityUpdated("u", "political", "")
	(&NotificationClient{}).EmitIdentityUpdated("u", "political", "")

	done := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("down"))
		done <- struct{}{}
	}))
	defer srv.Close()
	NewNotificationClient(srv.URL, "secret").EmitIdentityUpdated("u", "public_figure", "")
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("requête erreur non reçue")
	}

	(&NotificationClient{url: "%", client: http.DefaultClient}).EmitIdentityUpdated("u", "political", "")
}
