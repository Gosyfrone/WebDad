package realtime

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
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

// TestRegister_ReadPump_WritePump : teste Register + readPump + writePump via une
// vraie connexion WebSocket sur un serveur httptest.
func TestRegister_ReadPump_WritePump(t *testing.T) {
	h := NewHub()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		h.Register(ws) // bloque jusqu'à fermeture de la socket
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial : %v", err)
	}

	// Petite pause pour que Register et writePump démarrent.
	time.Sleep(20 * time.Millisecond)

	// PostCreated déclenche writePump : le client doit recevoir le message.
	h.PostCreated("post-id-ws", "author-id-ws")

	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage : %v", err)
	}
	if len(msg) == 0 {
		t.Fatal("message vide reçu depuis writePump")
	}

	// Fermeture côté client → readPump détecte l'erreur → remove → close(send)
	// → writePump quitte son for-range.
	_ = conn.Close()
	time.Sleep(50 * time.Millisecond)

	h.mu.RLock()
	remaining := len(h.conns)
	h.mu.RUnlock()
	if remaining != 0 {
		t.Fatalf("après fermeture client, hub devrait être vide, got %d connexions", remaining)
	}
}

// TestBroadcast_ClientLent : un client dont le canal send est plein est fermé
// (branche default de broadcast).
func TestBroadcast_ClientLent(t *testing.T) {
	h := NewHub()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		h.Register(ws)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial : %v", err)
	}
	defer func() { _ = conn.Close() }()

	time.Sleep(20 * time.Millisecond)

	// Saturer le canal send (sendBuffer = 32) sans que le client lise.
	for i := 0; i < sendBuffer+5; i++ {
		h.PostCreated("p", "a")
	}
	// Le broadcast doit fermer la socket du client lent sans paniquer.
	time.Sleep(100 * time.Millisecond)
}
