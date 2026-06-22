package realtime

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

func TestRemove_ConnAbsentePourUtilisateurConnu(t *testing.T) {
	h := NewHub()
	h.mu.Lock()
	h.conns["u1"] = make(map[*Conn]struct{})
	registered := &Conn{hub: h, userID: "u1", send: make(chan []byte, 1)}
	h.conns["u1"][registered] = struct{}{}
	h.mu.Unlock()

	unknown := &Conn{hub: h, userID: "u1", send: make(chan []byte, 1)}
	h.remove(unknown)

	h.mu.RLock()
	_, exists := h.conns["u1"][registered]
	h.mu.RUnlock()
	if !exists {
		t.Fatal("la connexion enregistrée ne doit pas être retirée")
	}
}

func TestPublishToMultipleConnections(t *testing.T) {
	h := NewHub()
	first := &Conn{hub: h, userID: "u1", send: make(chan []byte, 1)}
	second := &Conn{hub: h, userID: "u1", send: make(chan []byte, 1)}
	h.conns["u1"] = map[*Conn]struct{}{first: {}, second: {}}

	h.Publish([]string{"u1", "absent"}, map[string]string{"type": "refresh"})

	for i, conn := range []*Conn{first, second} {
		select {
		case <-conn.send:
		default:
			t.Fatalf("connexion %d sans message", i)
		}
	}
}

func TestRegisterPublishesAndRemovesWebSocket(t *testing.T) {
	h := NewHub()
	done := make(chan struct{})
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		h.Register("user", ws)
		close(done)
	}))
	defer server.Close()

	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		h.mu.RLock()
		registered := len(h.conns["user"]) == 1
		h.mu.RUnlock()
		if registered {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("connexion non enregistrée")
		}
		time.Sleep(time.Millisecond)
	}
	h.Publish([]string{"user"}, map[string]string{"type": "notification"})
	_, payload, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if !strings.Contains(string(payload), "notification") {
		t.Fatalf("payload = %s", payload)
	}
	if err := client.WriteMessage(websocket.TextMessage, []byte("ignored")); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Register ne s'est pas terminé")
	}
	h.mu.RLock()
	_, exists := h.conns["user"]
	h.mu.RUnlock()
	if exists {
		t.Fatal("connexion non retirée après fermeture")
	}
}

func TestPublishClosesSaturatedConnection(t *testing.T) {
	var serverWS *websocket.Conn
	ready := make(chan struct{})
	var once sync.Once
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverWS = ws
		once.Do(func() { close(ready) })
	}))
	defer server.Close()
	defer func() { _ = serverWS.Close() }()

	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer client.Close()
	<-ready

	h := NewHub()
	c := &Conn{hub: h, userID: "slow", ws: serverWS, send: make(chan []byte, 1)}
	c.send <- []byte("already full")
	h.conns["slow"] = map[*Conn]struct{}{c: {}}
	h.Publish([]string{"slow"}, map[string]string{"type": "overflow"})

	_ = client.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err := client.ReadMessage(); err == nil {
		t.Fatal("la connexion saturée aurait dû être fermée")
	}

	failedWriter := &Conn{ws: serverWS, send: make(chan []byte, 1)}
	failedWriter.send <- []byte("write after close")
	close(failedWriter.send)
	failedWriter.writePump()
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
