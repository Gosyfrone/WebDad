package client

import (
	"testing"
)

// ─── constantes ──────────────────────────────────────────────────────────────

func TestTypeConstants(t *testing.T) {
	if TypeFollow != "follow" {
		t.Fatalf("TypeFollow = %q", TypeFollow)
	}
	if TypeFollowRequest != "follow_request" {
		t.Fatalf("TypeFollowRequest = %q", TypeFollowRequest)
	}
	if TypeFollowRequestAccepted != "follow_request_accepted" {
		t.Fatalf("TypeFollowRequestAccepted = %q", TypeFollowRequestAccepted)
	}
	if TypeFollowRequestAcceptConfirm != "follow_request_accept_confirm" {
		t.Fatalf("TypeFollowRequestAcceptConfirm = %q", TypeFollowRequestAcceptConfirm)
	}
	if TypeFollowRequestRejected != "follow_request_rejected" {
		t.Fatalf("TypeFollowRequestRejected = %q", TypeFollowRequestRejected)
	}
}

// ─── NewNotificationClient ───────────────────────────────────────────────────

func TestNewNotificationClient_URLConstruite(t *testing.T) {
	c := NewNotificationClient("http://notification-service:8086", "shared-secret")
	want := "http://notification-service:8086/internal/events"
	if c.url != want {
		t.Fatalf("url = %q, attendu %q", c.url, want)
	}
	if c.secret != "shared-secret" {
		t.Fatalf("secret = %q, attendu 'shared-secret'", c.secret)
	}
	if c.client == nil {
		t.Fatal("http client nil")
	}
}

// ─── Emit ────────────────────────────────────────────────────────────────────

func TestEmit_NilClient_NePasPlanter(t *testing.T) {
	var c *NotificationClient
	c.Emit(Event{Type: TypeFollow, ActorID: "a", RecipientID: "b"})
}

func TestEmit_URLVide_NePasPlanter(t *testing.T) {
	c := &NotificationClient{url: "", secret: "s"}
	c.Emit(Event{Type: TypeFollow})
}
