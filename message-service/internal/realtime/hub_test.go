package realtime

import (
	"testing"
)

func TestNewHub(t *testing.T) {
	h := NewHub()
	if h == nil {
		t.Fatal("NewHub() = nil")
	}
}

func TestOnlineFrom_HubVide(t *testing.T) {
	h := NewHub()
	online := h.OnlineFrom([]string{"u1", "u2"})
	if len(online) != 0 {
		t.Fatalf("hub vide : OnlineFrom = %v, attendu []", online)
	}
}

func TestOnlineFrom_ListeVide(t *testing.T) {
	h := NewHub()
	online := h.OnlineFrom(nil)
	if len(online) != 0 {
		t.Fatalf("liste vide : OnlineFrom = %v, attendu []", online)
	}
}

func TestPublish_HubVide(t *testing.T) {
	h := NewHub()
	// Publish sans connexions ne doit pas paniquer
	h.Publish([]string{"u1"}, map[string]string{"type": "test"})
}

func TestPublish_ListeVide(t *testing.T) {
	h := NewHub()
	h.Publish(nil, map[string]string{"type": "test"})
}

func TestPublish_PayloadNonSérialisable(t *testing.T) {
	h := NewHub()
	// Un canal (non sérialisable en JSON) ne doit pas paniquer
	h.Publish([]string{"u1"}, make(chan int))
}

func TestOnlineFrom_ApresDeconnexion(t *testing.T) {
	h := NewHub()
	// Injecter une connexion artificielle dans le map pour tester OnlineFrom
	h.mu.Lock()
	h.conns["u1"] = make(map[*Conn]struct{})
	// Ajouter une conn factice
	fake := &Conn{hub: h, userID: "u1", send: make(chan []byte, 1)}
	h.conns["u1"][fake] = struct{}{}
	h.mu.Unlock()

	online := h.OnlineFrom([]string{"u1", "u2"})
	if len(online) != 1 || online[0] != "u1" {
		t.Fatalf("OnlineFrom = %v, attendu [u1]", online)
	}
}

func TestRemove_ConnInconnue(t *testing.T) {
	h := NewHub()
	// Appel remove sur une connexion non enregistrée ne doit pas paniquer
	fake := &Conn{hub: h, userID: "u99", send: make(chan []byte, 1)}
	h.remove(fake)
}
