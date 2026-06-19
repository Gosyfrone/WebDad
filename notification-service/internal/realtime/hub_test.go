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

func TestPublish_HubVide(t *testing.T) {
	h := NewHub()
	h.Publish([]string{"u1"}, map[string]string{"type": "test"})
}

func TestPublish_ListeVide(t *testing.T) {
	h := NewHub()
	h.Publish([]string{}, map[string]string{"type": "test"})
	h.Publish(nil, map[string]string{"type": "test"})
}

func TestPublish_PayloadNonSérialisable(t *testing.T) {
	h := NewHub()
	// Canal non sérialisable en JSON → ne doit pas paniquer
	h.Publish([]string{"u1"}, make(chan int))
}

func TestPublishToConnectedUser(t *testing.T) {
	h := NewHub()

	// Injecter une connexion factice
	h.mu.Lock()
	h.conns["u1"] = make(map[*Conn]struct{})
	fake := &Conn{hub: h, userID: "u1", send: make(chan []byte, 4)}
	h.conns["u1"][fake] = struct{}{}
	h.mu.Unlock()

	h.Publish([]string{"u1"}, map[string]string{"event": "new_notification"})

	// La connexion doit avoir reçu un message
	select {
	case msg := <-fake.send:
		if len(msg) == 0 {
			t.Fatal("message vide reçu")
		}
	default:
		t.Fatal("Publish n'a rien envoyé au canal")
	}
}

func TestRemove_ConnInconnue(t *testing.T) {
	h := NewHub()
	fake := &Conn{hub: h, userID: "unknown", send: make(chan []byte, 1)}
	h.remove(fake)
}

func TestRemove_VideLEntréeUtilisateur(t *testing.T) {
	h := NewHub()
	h.mu.Lock()
	h.conns["u2"] = make(map[*Conn]struct{})
	fake := &Conn{hub: h, userID: "u2", send: make(chan []byte, 1)}
	h.conns["u2"][fake] = struct{}{}
	h.mu.Unlock()

	h.remove(fake)

	h.mu.RLock()
	_, exists := h.conns["u2"]
	h.mu.RUnlock()
	if exists {
		t.Fatal("l'entrée u2 doit être supprimée quand la dernière connexion est retirée")
	}
}
