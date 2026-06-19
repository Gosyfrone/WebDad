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

func TestPostCreated_HubVide(t *testing.T) {
	h := NewHub()
	// PostCreated sans connexion ne doit pas paniquer
	h.PostCreated("post-id-123", "author-id-456")
}

func TestPostCreated_AvecConnexion(t *testing.T) {
	h := NewHub()

	// Injecter une connexion factice
	h.mu.Lock()
	fake := &Conn{hub: h, send: make(chan []byte, 4)}
	h.conns[fake] = struct{}{}
	h.mu.Unlock()

	h.PostCreated("post-id-123", "author-id-456")

	// La connexion doit avoir reçu un message
	select {
	case msg := <-fake.send:
		if len(msg) == 0 {
			t.Fatal("message vide reçu")
		}
	default:
		t.Fatal("PostCreated n'a rien envoyé au canal")
	}
}

func TestRemove_ConnInconnue(t *testing.T) {
	h := NewHub()
	fake := &Conn{hub: h, send: make(chan []byte, 1)}
	h.remove(fake)
}

func TestRemove_ConnEnregistrée(t *testing.T) {
	h := NewHub()
	h.mu.Lock()
	fake := &Conn{hub: h, send: make(chan []byte, 1)}
	h.conns[fake] = struct{}{}
	h.mu.Unlock()

	h.remove(fake)

	h.mu.RLock()
	_, still := h.conns[fake]
	h.mu.RUnlock()
	if still {
		t.Fatal("remove doit supprimer la connexion du hub")
	}
}
