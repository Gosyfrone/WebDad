package notifier

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNoop_Emit(t *testing.T) {
	n := Noop{}
	// Ne doit pas paniquer
	n.Emit(Event{Type: TypeMessage, ActorID: "a", RecipientID: "b", ConversationID: "c"})
}

func TestNew_ConstructHTTPNotifier(t *testing.T) {
	n := New("http://notif:8086", "secret")
	if n == nil {
		t.Fatal("New() = nil")
	}
	if n.url != "http://notif:8086/internal/events" {
		t.Fatalf("url = %q, attendu /internal/events", n.url)
	}
}

func TestHTTPNotifier_Emit_Success(t *testing.T) {
	received := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Get("X-Internal-Secret")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	n := New(srv.URL, "mysecret")
	n.Emit(Event{Type: TypeMessage, ActorID: "u1", RecipientID: "u2", ConversationID: "conv1"})

	// Le Emit est fire-and-forget en goroutine → on attend la réception
	select {
	case secret := <-received:
		if secret != "mysecret" {
			t.Fatalf("X-Internal-Secret = %q, attendu mysecret", secret)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveur n'a pas reçu la requête dans le délai imparti")
	}
}

func TestHTTPNotifier_Emit_ServerDown(t *testing.T) {
	// Serveur fermé immédiatement : l'Emit doit être silencieux (fire-and-forget)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	n := New(srv.URL, "secret")
	// Ne doit pas paniquer ni retourner d'erreur (fire-and-forget)
	n.Emit(Event{Type: TypeMessageMention, ActorID: "u1", RecipientID: "u2", ConversationID: "c1"})

	// Attendre que la goroutine se termine
	time.Sleep(100 * time.Millisecond)
}

func TestEventTypes(t *testing.T) {
	if TypeMessage == "" {
		t.Fatal("TypeMessage ne doit pas être vide")
	}
	if TypeMessageMention == "" {
		t.Fatal("TypeMessageMention ne doit pas être vide")
	}
}
