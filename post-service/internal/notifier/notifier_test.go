package notifier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestParseMentions(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{"aucune mention", "bonjour tout le monde", nil},
		{"une mention", "salut @bob !", []string{"bob"}},
		{"plusieurs mentions", "@alice et @bob_42 venez voir", []string{"alice", "bob_42"}},
		{"déduplication insensible à la casse", "@Bob @bob @BOB", []string{"Bob"}},
		{"email non capté comme mention", "écris à jean@exemple.com", nil},
		{"handle trop court ignoré", "@ab @bob", []string{"bob"}},
		{"point interne capturé", "coucou @jean.dupont", []string{"jean.dupont"}},
		{"point final exclu", "salut @bob.", []string{"bob"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseMentions(tc.content)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParseMentions(%q) = %v ; attendu %v", tc.content, got, tc.want)
			}
		})
	}
}

// TestNoop : le no-op n'émet rien et ne panique pas (post-service autonome).
func TestNoop(t *testing.T) {
	var n Notifier = Noop{}
	n.Emit(Event{Type: TypeLike}) // ne doit rien faire
}

// TestNew : constructeur HTTPNotifier.
func TestNew(t *testing.T) {
	n := New("http://notification-service:8086", "mysecret")
	if n == nil {
		t.Fatal("New() = nil")
	}
	if n.url != "http://notification-service:8086/internal/events" {
		t.Fatalf("New().url = %q, attendu suffix /internal/events", n.url)
	}
}

// TestHTTPNotifier_Emit_Success : vérifie qu'un événement est bien posté.
func TestHTTPNotifier_Emit_Success(t *testing.T) {
	received := make(chan Event, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ev Event
		if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		received <- ev
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := New(srv.URL, "secret")
	n.Emit(Event{Type: TypeLike, ActorID: "alice", RecipientID: "bob"})

	select {
	case ev := <-received:
		if ev.Type != TypeLike {
			t.Fatalf("type reçu = %q, attendu %q", ev.Type, TypeLike)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("événement non reçu dans le délai imparti")
	}
}

// TestHTTPNotifier_Emit_ServeDown : le notifier ne panique pas si le serveur est indisponible.
func TestHTTPNotifier_Emit_ServeDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	n := New(url, "secret")
	n.Emit(Event{Type: TypeComment}) // ne doit pas paniquer ni bloquer
	time.Sleep(100 * time.Millisecond)
}
